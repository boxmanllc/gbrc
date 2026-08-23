package main

import (
	"flag"
	"fmt"
	"log"

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

	fmt.Println(rom)
	d := decoder.NewDecoder(rom)

	for instr := range d.Decode() {
		if instr.InstructionType == decoder.UNKNOWN {
			log.Fatalf("found unknown instruction at 0x%04X", instr.Address)
		} else {
			fmt.Printf("%+v\n", instr)
		}
	}
}
