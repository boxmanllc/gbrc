package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

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
	instrs := decoder.Decode()

	for _, instr := range instrs {
		fmt.Printf("%+v\n", instr)
		fmt.Println(strings.Repeat("=", 10))
	}
}
