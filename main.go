package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/0xmukesh/boxman/internal/analyzer"
	"github.com/0xmukesh/boxman/internal/codegen"
	"github.com/0xmukesh/boxman/internal/decoder"
	"github.com/0xmukesh/boxman/internal/rom"
)

var (
	romFilePath string
	irFilePath  string
)

func main() {
	flag.StringVar(&romFilePath, "rom", "", "path where rom file is present")
	flag.StringVar(&irFilePath, "ir", "", "path where llvm ir would be saved")
	flag.Parse()
	validateFlags()

	rom, err := rom.Parse(romFilePath)
	if err != nil {
		log.Fatalf("failed to parse rom file: %s", err)
	}

	decoder := decoder.NewDecoder(rom)
	analyzer := analyzer.NewAnalyzer(decoder)
	codegen := codegen.NewCodegen()

	blocks := analyzer.AnalyzeBlocks()

	for _, block := range blocks {
		fmt.Printf("%04X-%04X\n", block.Start, block.End)
		for _, s := range block.Successors {
			fmt.Printf("%04X\n", s)
		}
		fmt.Println(strings.Repeat("=", 10))
	}

	os.WriteFile(irFilePath, []byte(codegen.IR()), 0644)
}

func validateFlags() {
	if romFilePath == "" {
		log.Fatalf("missing rom file path")
	}

	if irFilePath == "" {
		irFilePath = strings.TrimSuffix(romFilePath, filepath.Ext(romFilePath)) + ".ll"
	}
}
