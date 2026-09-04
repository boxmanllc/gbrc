#include "ram.h"
#include "gb.h"
#include "joypad.h"
#include <stdint.h>

uint8_t read_ram(uint16_t addr) {
	switch (addr) {
	case 0xFF00:
		return joypad_read_reg();
	default:
		return ram[addr];
	}
}
void write_ram(uint16_t addr, uint16_t val) { ram[addr] = val; }
