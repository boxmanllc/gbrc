package cmd

import (
	"flag"
	"log"
	"path/filepath"
	"strings"

	"github.com/0xmukesh/boxman/internal/analyzer"
	"github.com/0xmukesh/boxman/internal/codegen"
	"github.com/0xmukesh/boxman/internal/decoder"
	"github.com/0xmukesh/boxman/internal/rom"
)

var (
	romFilePath, irFilePath string
)

func Run() {
	parseFlags()

	rom, err := rom.Parse(romFilePath)
	if err != nil {
		log.Fatalf("failed to parse rom file: %s", err)
	}

	decoder := decoder.NewDecoder(rom)
	analyzer := analyzer.NewAnalyzer(decoder)
	blocks := analyzer.AnalyzeBlocks()

	codegen, err := codegen.NewCodegen(blocks)
	if err != nil {
		log.Fatalf("failed to codegen: %s", err)
	}

	codegen.WriteTo(irFilePath)
}

func parseFlags() {
	flag.StringVar(&romFilePath, "rom", "", "path where rom file is present")
	flag.StringVar(&irFilePath, "ir", "", "path where llvm ir should be saved to")
	flag.Parse()

	if romFilePath == "" {
		log.Fatalf("missing rom file path")
	}

	if irFilePath == "" {
		irFilePath = strings.TrimSuffix(romFilePath, filepath.Ext(romFilePath)) + ".ll"
	}
}
