package rom

import "os"

type Rom struct {
	data []byte
}

func Parse(romFilePath string) (*Rom, error) {
	data, err := os.ReadFile(romFilePath)
	if err != nil {
		return nil, err
	}

	return &Rom{data: data}, nil
}

func (r *Rom) Read(addr uint16) uint8 {
	if int(addr) >= len(r.data) {
		return 0
	}

	return r.data[addr]
}

func (r *Rom) Bytes() []byte {
	return r.data
}
