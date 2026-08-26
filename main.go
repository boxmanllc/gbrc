package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/0xmukesh/boxman/internal/analyzer"
	"github.com/0xmukesh/boxman/internal/decoder"
	"github.com/0xmukesh/boxman/internal/rom"
)

var (
	romFilePath string
)

func main() {
	flag.StringVar(&romFilePath, "rom", "", "path to the rom file")
	flag.Parse()

	rom, err := rom.Parse(romFilePath)
	if err != nil {
		log.Fatalf("failed to parse rom file: %s", err)
	}

	decoder := decoder.NewDecoder(rom)
	analyzer := analyzer.NewAnalyzer(decoder)

	for addr := range analyzer.FindReachableAddresses() {
		fmt.Printf("0x%04X\n", addr)
	}
}
