#include "joypad.h"
#include <stdbool.h>
#include <stdint.h>
#include <string.h>

const Button DPAD_BUTTONS[4] = {RIGHT, LEFT, UP, DOWN};
const Button FACE_BUTTONS[4] = {A, B, SELECT, START};

void new_pad(Jp *pad) {
	memset(pad->buttons, false, sizeof(pad->buttons));
	pad->dpad_selected = false;
	pad->face_selected = false;
	memset(pad->ram, 0, sizeof(pad->ram));
}

uint8_t read_u8(Jp *pad, uint16_t addr) {
	if (addr == JOYPAD_ADDR) {
		return read_joypad(pad);
	} else {
		uint16_t rel = addr - IO_START;
		if (rel < IO_SIZE) {
			return pad->ram[rel];
		} else {
			return 0xFF;
		}
	}
}

uint8_t read_joypad(Jp *pad) {
	uint8_t ret = 0x0F; // 0b00_00_1111

	if (pad->dpad_selected) {
		for (int i = 0; i < 4; ++i) {
			int idx = (int)DPAD_BUTTONS[i];
			if (pad->buttons[idx])
				// idx & 3 trims it to lower 2 bits
				// 5: 0b101
				// 3: 0b011
				// 5 & 3 => 0b001 => 1
				ret &= ~(1u << (idx & 3));
		}
	}

	if (pad->face_selected) {
		for (int i = 0; i < 4; ++i) {
			int idx = (int)FACE_BUTTONS[i];
			if (pad->buttons[idx])
				ret &= ~(1u << (idx & 3));
		}
	}
	// 0xC0 = 0b11_00_0000, unused bits 6 and 7 always 1
	ret |= 0xC0;
	return ret;
}

void set_button(Jp *pad, Button button, bool pressed) {
	pad->buttons[button] = pressed;
}

void write_u8(Jp *pad, uint16_t addr, uint8_t val) {
	if (addr == JOYPAD_ADDR) {
		// get bit at FACE_SELECT_BIT in val
		uint8_t face_bit = (val >> FACE_SELECT_BIT) & 1u;
		// get bit at DPAD_SELECT_BIT in val
		uint8_t dpad_bit = (val >> DPAD_SELECT_BIT) & 1u;
		pad->face_selected = !face_bit;
		pad->dpad_selected = !dpad_bit;
	} else {
		uint16_t rel = addr - IO_START;
		if (rel < IO_SIZE)
			pad->ram[rel] = val;
	}
}
