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
		if _, err = exec.Command("opt", "-O2", "-S", irFilePath, "-o", irFilePath).Output(); err != nil {
			log.Fatalf("failed to optimize ir: %s", err)
		}
	}

	if toCompile {
		projectRoot, err := findProjectRoot()
		if err != nil {
			log.Fatalf("failed to locate project root: %s", err)
		}

		runtimeInclude := filepath.Join(projectRoot, "runtime", "include")
		mainC := filepath.Join(projectRoot, "runtime", "src", "main.c")
		ramC := filepath.Join(projectRoot, "runtime", "src", "ram.c")
		joypadC := filepath.Join(projectRoot, "runtime", "src", "joypad.c")
		interruptC := filepath.Join(projectRoot, "runtime", "src", "interrupt.c")

		if _, err := exec.Command("clang", "-O0", "-g",
			"-I"+runtimeInclude,
			irFilePath,
			mainC, ramC, joypadC, interruptC,
			"-o", outFilePath).Output(); err != nil {
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
