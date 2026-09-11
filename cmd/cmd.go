package cmd

import (
	"flag"
	"log"
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
		if _, err := exec.Command("clang", "-O0", "-g",
			"-Iruntime/include",
			irFilePath,
			"runtime/src/main.c", "runtime/src/ram.c",
			"runtime/src/joypad.c", "runtime/src/interrupt.c",
			"-o", outFilePath).Output(); err != nil {
			log.Fatalf("failed to compile ir: %s", err)
		}
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
