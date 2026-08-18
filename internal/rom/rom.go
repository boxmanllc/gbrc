package rom

import (
	"fmt"
	"os"
)

type CartridgeType int

var (
	RomOnly        CartridgeType = 0x00
	MBC1           CartridgeType = 0x01
	MBC1RAM        CartridgeType = 0x02
	MBC1RAMBattery CartridgeType = 0x03
	MBC2           CartridgeType = 0x05
	MBC2Battery    CartridgeType = 0x06

	CartridgeTypeNameMapping = map[CartridgeType]string{
		RomOnly:        "ROM ONLY",
		MBC1:           "MBC1",
		MBC1RAM:        "MBC1+RAM",
		MBC1RAMBattery: "MBC1+RAM+BATTERY",
		MBC2:           "MBC2",
		MBC2Battery:    "MBC2+BATTERY",
	}
)

type Rom struct {
	Title         string
	Cgb           bool
	CartridgeType CartridgeType
	RomSize       int
	RamSize       int
	RomVersion    int

	data []uint8
}

func (r *Rom) String() string {
	return fmt.Sprintf(`ROM Information:
  Title: %s
  MBC Type: %s
  ROM Size: %.2f KiB
  RAM Size: %.2f KiB
  CGB Support: %t
  ROM Version: %d`,
		r.Title, CartridgeTypeNameMapping[r.CartridgeType],
		(float64(r.RomSize) / 1024), (float64(r.RamSize) / 1024),
		r.Cgb, r.RomVersion,
	)
}

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

	romVersion := int(bytes[0x014C])

	return &Rom{
		Title:         title,
		Cgb:           cgb,
		CartridgeType: cartridgeType,
		RomSize:       romSize,
		RamSize:       ramSize,
		RomVersion:    romVersion,
		data:          bytes,
	}, nil
}

func (r *Rom) Read(addr uint16) uint8 {
	return r.data[addr]
}
