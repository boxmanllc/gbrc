#include "ram.h"
#include "gb.h"
#include "joypad.h"
#include <stdint.h>

uint8_t read_ram(uint16_t addr) {
	switch (addr) {
	case 0xFF00:
		return joypad_read();
	default:
		return ram[addr];
	}
}
void write_ram(uint16_t addr, uint8_t val) {
	switch (addr) {
	case 0xFF00:
		joypad_write(val);
		ram[addr] = val;
		return;
	default:
		ram[addr] = val;
	}
}
