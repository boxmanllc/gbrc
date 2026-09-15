package cmd

import (
	"bufio"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/boxmanllc/gbrc/internal/analyzer"
	"github.com/boxmanllc/gbrc/internal/codegen"
	"github.com/boxmanllc/gbrc/internal/decoder"
	"github.com/boxmanllc/gbrc/internal/rom"
)

var (
	romFilePath, outDir   string
	profileFilePath       string
	toOptimize, toCompile bool
)

func Run(runtimeFS fs.FS) {
	parseFlags()

	romFile, err := rom.Parse(romFilePath)
	if err != nil {
		log.Fatalf("failed to parse rom file: %s", err)
	}

	decoder := decoder.New(romFile)
	analyzer := analyzer.New(decoder)

	if profileFilePath != "" {
		seeds, err := readSeeds(profileFilePath)
		if err != nil {
			log.Fatalf("failed to read profile: %s", err)
		}

		analyzer.ExtraSeeds = seeds
		log.Printf("profile: %d extra seed(s) from %s", len(seeds), profileFilePath)
	}

	blocks := analyzer.AnalyzeBlocks()
	log.Printf("discovered %d basic block(s)", len(blocks))

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Fatalf("failed to create output directory: %s", err)
	}

	name := romBaseName(romFilePath)
	irPath := filepath.Join(outDir, name+".ll")
	binPath := filepath.Join(outDir, name)

	cg, err := codegen.New(blocks, romFile.Bytes())
	if err != nil {
		log.Fatalf("failed to generate ir: %s", err)
	}

	if err := cg.WriteTo(irPath); err != nil {
		log.Fatalf("failed to write ir: %s", err)
	}

	if toOptimize {
		requireTool("opt")
		optCmd := exec.Command("opt", "-O2", "-S", irPath, "-o", irPath)
		optCmd.Stderr = os.Stderr
		if err := optCmd.Run(); err != nil {
			log.Fatalf("failed to optimize ir: %s", err)
		}
	}

	log.Printf("wrote %s", irPath)

	if toCompile {
		if err := compile(runtimeFS, irPath, binPath); err != nil {
			log.Fatalf("%s", err)
		}

		log.Printf("wrote %s", binPath)
	}
}

func parseFlags() {
	flag.Usage = usage

	flag.StringVar(&romFilePath, "rom", "", "path to the gameboy rom")
	flag.StringVar(&outDir, "out", "", "output directory for the llvm ir and binary (default: directory of the rom)")
	flag.StringVar(&profileFilePath, "profile", "", "seed block discovery from a profile of interpreted entry points")
	noOptFlag := flag.Bool("no-optimize", false, "skip llvm ir optimization")
	noCompileFlag := flag.Bool("no-compile", false, "emit llvm ir only")
	flag.Parse()

	if romFilePath == "" {
		usage()
		os.Exit(1)
	}

	if _, err := os.Stat(romFilePath); err != nil {
		log.Fatalf("cannot read rom: %s", err)
	}

	if outDir == "" {
		outDir = filepath.Dir(romFilePath)
	}

	toOptimize = !*noOptFlag
	toCompile = !*noCompileFlag
}

func usage() {
	out := flag.CommandLine.Output()
	fmt.Fprintf(out, "gbrc - gameboy static recompiler\n\n")
	fmt.Fprintf(out, "usage:\n  gbrc -rom <rom.gb> [options]\n\n")
	fmt.Fprintf(out, "options:\n")
	flag.PrintDefaults()
}

func romBaseName(path string) string {
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

func readSeeds(path string) ([]uint16, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	seen := make(map[uint16]bool)
	var seeds []uint16

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		v, err := strconv.ParseUint(line, 16, 16)
		if err != nil {
			continue
		}

		addr := uint16(v)
		if addr < analyzer.ROM_END && !seen[addr] {
			seen[addr] = true
			seeds = append(seeds, addr)
		}
	}

	return seeds, scanner.Err()
}

func compile(runtimeFS fs.FS, irPath, outPath string) error {
	runtimeDir, cleanup, err := extractRuntime(runtimeFS)
	if err != nil {
		return fmt.Errorf("failed to unpack runtime: %w", err)
	}
	defer cleanup()

	requireTool("clang")
	requireTool("pkg-config")

	runtimeInclude := filepath.Join(runtimeDir, "include")
	runtimeSources, err := filepath.Glob(filepath.Join(runtimeDir, "src", "*.c"))
	if err != nil || len(runtimeSources) == 0 {
		return fmt.Errorf("failed to locate runtime sources: %w", err)
	}

	sdlCFlags, err := exec.Command("pkg-config", "--cflags", "sdl2").Output()
	if err != nil {
		return fmt.Errorf("sdl2 not found")
	}

	sdlLibs, err := exec.Command("pkg-config", "--libs", "sdl2").Output()
	if err != nil {
		return fmt.Errorf("sdl2 not found")
	}

	args := []string{"-O0", "-g", "-I" + runtimeInclude}
	args = append(args, strings.Fields(string(sdlCFlags))...)
	args = append(args, irPath)
	args = append(args, runtimeSources...)
	args = append(args, strings.Fields(string(sdlLibs))...)
	args = append(args, "-o", outPath)

	compileCmd := exec.Command("clang", args...)
	compileCmd.Stderr = os.Stderr
	if err := compileCmd.Run(); err != nil {
		return fmt.Errorf("failed to compile ir: %w", err)
	}

	return nil
}

func extractRuntime(runtimeFS fs.FS) (string, func(), error) {
	dir, err := os.MkdirTemp("", "gbrc-runtime-")
	if err != nil {
		return "", nil, err
	}

	cleanup := func() { os.RemoveAll(dir) }

	if err = fs.WalkDir(runtimeFS, "runtime", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		rel := strings.TrimPrefix(path, "runtime/")
		target := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}

		data, err := fs.ReadFile(runtimeFS, path)
		if err != nil {
			return err
		}

		return os.WriteFile(target, data, 0o644)
	}); err != nil {
		cleanup()
		return "", nil, err
	}

	return dir, cleanup, nil
}

func requireTool(name string) {
	if _, err := exec.LookPath(name); err != nil {
		log.Fatalf("required tool %q not found in PATH", name)
	}
}
