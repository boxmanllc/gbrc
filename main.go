package main

import (
	"flag"
	"log"

	"github.com/0xmukesh/boxman/internal/rom"
)

var (
	romFilePath string
)

func main() {
	flag.StringVar(&romFilePath, "rom", "", "path to the rom file")
	flag.Parse()

	_, err := rom.Parse(romFilePath)
	if err != nil {
		log.Fatalf("failed to parse rom file: %s", err)
	}
}
