package cmd

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/0xmukesh/boxman/internal/analyzer"
	"github.com/0xmukesh/boxman/internal/codegen"
	"github.com/0xmukesh/boxman/internal/decoder"
	"github.com/0xmukesh/boxman/internal/rom"
)

var (
	romFilePath, irFilePath, outFilePath string
	debugFlag                            bool
	toOptimize, toCompile                bool
)

func Run() {
	parseFlags()

	romFile, err := rom.Parse(romFilePath)
	if err != nil {
		log.Fatalf("failed to parse rom file: %s", err)
	}

	dec := decoder.New(romFile)
	an := analyzer.New(dec)
	blocks := an.AnalyzeBlocks()

	cg, err := codegen.New(blocks, romFile.Bytes(), debugFlag)
	if err != nil {
		log.Fatalf("failed to generate ir: %s", err)
	}

	if err := cg.WriteTo(irFilePath); err != nil {
		log.Fatalf("failed to write ir: %s", err)
	}

	if toOptimize {
		optCmd := exec.Command("opt", "-O2", "-S", irFilePath, "-o", irFilePath)
		optCmd.Stderr = os.Stderr
		if err := optCmd.Run(); err != nil {
			log.Fatalf("failed to optimize ir: %s", err)
		}
	}

	if toCompile {
		projectRoot, err := findProjectRoot()
		if err != nil {
			log.Fatalf("failed to locate project root: %s", err)
		}

		runtimeInclude := filepath.Join(projectRoot, "runtime", "include")
		runtimeSources, err := filepath.Glob(filepath.Join(projectRoot, "runtime", "src", "*.c"))
		if err != nil || len(runtimeSources) == 0 {
			log.Fatalf("failed to locate runtime sources: %s", err)
		}

		sdlCFlagsOut, err := exec.Command("pkg-config", "--cflags", "sdl2").Output()
		if err != nil {
			log.Fatalf("sdl2 not found: pkg-config --cflags sdl2 failed: %s", err)
		}
		sdlLibsOut, err := exec.Command("pkg-config", "--libs", "sdl2").Output()
		if err != nil {
			log.Fatalf("sdl2 not found: pkg-config --libs sdl2 failed: %s", err)
		}

		args := []string{"-O0", "-g", "-I" + runtimeInclude}
		args = append(args, strings.Fields(string(sdlCFlagsOut))...)
		args = append(args, irFilePath)
		args = append(args, runtimeSources...)
		args = append(args, strings.Fields(string(sdlLibsOut))...)
		args = append(args, "-o", outFilePath)

		compileCmd := exec.Command("clang", args...)
		compileCmd.Stderr = os.Stderr
		if err := compileCmd.Run(); err != nil {
			log.Fatalf("failed to compile ir: %s", err)
		}
	}
}

func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

func parseFlags() {
	flag.StringVar(&romFilePath, "rom", "", "path where rom file is present")
	flag.StringVar(&irFilePath, "ir", "", "path where llvm ir should be saved to")
	flag.StringVar(&outFilePath, "out", "", "path where output binary would be saved to")
	flag.BoolVar(&debugFlag, "debug", false, "print out register, flags and cycles after executation of every block")
	noOptFlag := flag.Bool("no-optimize", false, "disable ir optimization")
	noCompileFlag := flag.Bool("no-compile", false, "emit ir only")
	flag.Parse()

	if romFilePath == "" {
		log.Fatalf("missing rom file path")
	}

	base := strings.TrimSuffix(romFilePath, filepath.Ext(romFilePath))

	if irFilePath == "" {
		irFilePath = base + ".ll"
	}

	if outFilePath == "" {
		outFilePath = base
	}

	toOptimize = !*noOptFlag
	toCompile = !*noCompileFlag
}
