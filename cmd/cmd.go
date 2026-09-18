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
	"slices"
	"strconv"
	"strings"

	"github.com/boxmanllc/gbrc/internal/analyzer"
	"github.com/boxmanllc/gbrc/internal/codegen"
	"github.com/boxmanllc/gbrc/internal/decoder"
	"github.com/boxmanllc/gbrc/internal/rom"
)

type options struct {
	romPath  string
	outDir   string
	profile  string
	frontend string
	target   string
	onlyIR   bool
	saveIR   bool
}

type toolchain struct {
	cc     string
	flags  []string
	libs   []string
	link   []string
	outExt string
}

var compileFlags = []string{
	"-O2",
	"-ffunction-sections",
	"-fdata-sections",
	"-fno-asynchronous-unwind-tables",
	"-fno-unwind-tables",
	"-fno-stack-protector",
}

func Run(runtimeFS fs.FS) {
	opt := parseFlags()

	romFile, err := rom.Parse(opt.romPath)
	must(err, "parse rom")

	an := analyzer.New(decoder.New(romFile))
	if opt.profile != "" {
		seeds, err := readSeeds(opt.profile)
		must(err, "read profile")

		an.ExtraSeeds = seeds
		log.Printf("profile: %d extra seed(s) from %s", len(seeds), opt.profile)
	}

	blocks := an.AnalyzeBlocks()
	log.Printf("discovered %d basic block(s)", len(blocks))

	must(os.MkdirAll(opt.outDir, 0o755), "create output directory")

	work, err := os.MkdirTemp("", "gbrc-")
	must(err, "create work directory")
	defer os.RemoveAll(work)

	cg, err := codegen.New(blocks, romFile.Bytes())
	must(err, "generate ir")

	name := strings.TrimSuffix(filepath.Base(opt.romPath), filepath.Ext(opt.romPath))
	ir := filepath.Join(work, name+".ll")
	must(cg.WriteTo(ir), "write ir")

	must(run("opt", "-O2", "-S", ir, "-o", ir), "optimize ir")

	if opt.onlyIR || opt.saveIR {
		data, err := os.ReadFile(ir)
		must(err, "read ir")
		dst := filepath.Join(opt.outDir, name+".ll")
		must(os.WriteFile(dst, data, 0o644), "save ir")
		log.Printf("wrote %s", dst)
	}
	if opt.onlyIR {
		return
	}

	tc := selectToolchain(opt.target)

	runtimeDir, err := extractRuntime(runtimeFS)
	must(err, "unpack runtime")
	defer os.RemoveAll(runtimeDir)

	include := filepath.Join(runtimeDir, "runtime", "include")

	romObj := filepath.Join(work, name+".o")
	compile(tc, ir, romObj, "-x", "ir")

	sources, err := collectSources(filepath.Join(runtimeDir, "runtime", "src"))
	must(err, "collect runtime sources")

	objs := []string{romObj}
	for i, src := range sources {
		obj := filepath.Join(work, fmt.Sprintf("core_%d.o", i))
		compile(tc, src, obj, "-I"+include)
		objs = append(objs, obj)
	}

	outPath := filepath.Join(opt.outDir, name+tc.outExt)
	args := slices.Clone(tc.flags)
	args = append(args, "-O2")
	args = append(args, objs...)
	args = append(args, opt.frontend)
	args = append(args, tc.libs...)
	args = append(args, tc.link...)
	args = append(args, "-o", outPath)

	must(run(tc.cc, args...), "link")
	log.Printf("wrote %s", outPath)
}

func parseFlags() options {
	var o options

	flag.StringVar(&o.romPath, "rom", "", "path to the gameboy rom (required)")
	flag.StringVar(&o.outDir, "out", "", "output directory (default: directory of the rom)")
	flag.StringVar(&o.profile, "profile", "", "seed additional entry points from a dynamic block discovery profile")
	flag.StringVar(&o.frontend, "frontend", "", "compiled object file of the frontend to link against (required)")
	flag.StringVar(&o.target, "target", "", "target architecture, e.g. wasm (uses emscripten for wasm targets)")
	flag.BoolVar(&o.onlyIR, "only-ir", false, "emit only the generated llvm ir")
	flag.BoolVar(&o.saveIR, "save-ir", false, "also emit the generated llvm ir alongside the binary")
	flag.Parse()

	if o.romPath == "" {
		flag.Usage()
		os.Exit(1)
	}

	_, err := os.Stat(o.romPath)
	must(err, "cannot read rom")

	if !o.onlyIR {
		if o.frontend == "" {
			log.Fatal("--frontend is required")
		}
		_, err = os.Stat(o.frontend)
		must(err, "cannot read frontend object")
	}

	if o.outDir == "" {
		o.outDir = filepath.Dir(o.romPath)
	}

	return o
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
		v, err := strconv.ParseUint(strings.TrimSpace(scanner.Text()), 16, 16)
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

func selectToolchain(target string) toolchain {
	switch {
	case target == "":
		tc := toolchain{
			cc: "clang",
			link: []string{
				"-Wl,--gc-sections",
				"-Wl,--build-id=none",
				"-Wl,-O1",
				"-Wl,-z,noseparate-code",
				"-s",
			},
		}

		if out, err := exec.Command("pkg-config", "--libs", "sdl2").Output(); err == nil {
			tc.libs = strings.Fields(string(out))
		}

		return tc
	case strings.HasPrefix(target, "wasm"):
		tc := toolchain{cc: "emcc", outExt: ".html"}
		if shell := os.Getenv("GBRC_SHELL"); shell != "" {
			tc.link = []string{"--shell-file", shell}
		}

		return tc
	default:
		return toolchain{cc: "clang", flags: []string{"-target", target}}
	}
}

func extractRuntime(runtimeFS fs.FS) (string, error) {
	dir, err := os.MkdirTemp("", "gbrc-runtime-")
	if err != nil {
		return "", err
	}

	err = fs.WalkDir(runtimeFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		target := filepath.Join(dir, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}

		data, err := fs.ReadFile(runtimeFS, path)
		if err != nil {
			return err
		}

		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		os.RemoveAll(dir)
		return "", err
	}

	return dir, nil
}

func collectSources(root string) ([]string, error) {
	var sources []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ".c" {
			sources = append(sources, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(sources) == 0 {
		return nil, fmt.Errorf("failed to locate runtime sources in %s", root)
	}

	return sources, nil
}

func compile(tc toolchain, src, obj string, extra ...string) {
	args := slices.Clone(tc.flags)
	args = append(args, compileFlags...)
	args = append(args, extra...)
	args = append(args, "-c", src, "-o", obj)
	must(run(tc.cc, args...), "compile "+filepath.Base(src))
}

func run(tool string, args ...string) error {
	cmd := exec.Command(tool, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func must(err error, what string) {
	if err != nil {
		log.Fatalf("%s: %s", what, err)
	}
}
