package rom

import (
	"os"
)

type CartridgeType int

type Rom struct {
	Title         string
	Cgb           bool
	CartridgeType CartridgeType
	RomSize       int
	RamSize       int
}

var (
	RomOnly        CartridgeType = 0x00
	MBC1           CartridgeType = 0x01
	MBC1RAM        CartridgeType = 0x02
	MBC1RAMBattery CartridgeType = 0x03
	MBC2           CartridgeType = 0x05
	MBC2Battery    CartridgeType = 0x06
)

func Parse(romFilePath string) (*Rom, error) {
	bytes, err := os.ReadFile(romFilePath)
	if err != nil {
		return nil, err
	}

	title := string(bytes[0x0134:0x0144])
	cgb := int(bytes[0x0143])&(1<<7) != 0
	cartridgeType := CartridgeType(bytes[0x0147])
	romSize := 32 * 1024 * (1 << int(bytes[0x0148]))
	ramSizeByte := int(bytes[0x0149])

	ramSize := 0
	switch ramSizeByte {
	case 0x00:
		ramSize = 0
	case 0x02:
		ramSize = 8 * 1024
	case 0x03:
		ramSize = 32 * 1024
	case 0x04:
		ramSize = 128 * 1024
	case 0x05:
		ramSize = 64 * 1024
	}

	return &Rom{
		Title:         title,
		Cgb:           cgb,
		CartridgeType: cartridgeType,
		RomSize:       romSize,
		RamSize:       ramSize,
	}, nil
}
