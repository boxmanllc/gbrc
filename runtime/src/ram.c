#include "ram.h"
#include "gb.h"
#include "joypad.h"
#include <stdint.h>

uint8_t read_ram(uint16_t addr) {
	switch (addr) {
	case 0xFF00:
		return joypad_read_reg();
	case 0xFF44:
		// LY (no PPU emulation yet): advance one scanline per 456 t-cycles,
		// 154 scanlines per frame, so vblank sync waits (CP 0x90..0x99)
		// always terminate.
		return (cycles / 456) % 154;
	default:
		return ram[addr];
	}
}
void write_ram(uint16_t addr, uint8_t val) {
	switch (addr) {
	case 0xFF00:
		joypad_write_reg(val);
		ram[addr] = val;
		return;
	default:
		ram[addr] = val;
	}
}
