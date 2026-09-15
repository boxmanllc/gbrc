#include "ram.h"
#include "gb.h"
#include "joypad.h"
#include "ppu.h"
#include "timer.h"
#include <stdint.h>

uint8_t read_ram(uint16_t addr) {
	switch (addr) {
	case 0xFF00:
		return joypad_read();
	case 0xFF04:
	case 0xFF05:
	case 0xFF06:
	case 0xFF07:
		return timer_read(addr);
	case 0xFF40:
	case 0xFF41:
	case 0xFF42:
	case 0xFF43:
	case 0xFF44:
	case 0xFF45:
	case 0xFF46:
	case 0xFF47:
	case 0xFF48:
	case 0xFF49:
	case 0xFF4A:
	case 0xFF4B:
		return ppu_read(addr);
	default:
		return ram[addr];
	}
}
void write_ram(uint16_t addr, uint8_t val) {
	switch (addr) {
	case 0xFF00:
		joypad_write(val);
		return;
	case 0xFF04:
	case 0xFF05:
	case 0xFF06:
	case 0xFF07:
		timer_write(addr, val);
		return;
	case 0xFF40:
	case 0xFF41:
	case 0xFF42:
	case 0xFF43:
	case 0xFF44:
	case 0xFF45:
	case 0xFF46:
	case 0xFF47:
	case 0xFF48:
	case 0xFF49:
	case 0xFF4A:
	case 0xFF4B:
		ppu_write(addr, val);
		return;
	default:
		ram[addr] = val;
	}
}
