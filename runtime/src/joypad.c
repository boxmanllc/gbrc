#include "joypad.h"
#include <stdbool.h>
#include <stdint.h>
#include <string.h>

const Button DPAD_BUTTONS[4] = {RIGHT, LEFT, UP, DOWN};
const Button FACE_BUTTONS[4] = {A, B, SELECT, START};

static Jp joypad;

void joypad_init_impl(Jp *pad) {
	memset(pad->buttons, false, sizeof(pad->buttons));
	pad->dpad_selected = false;
	pad->face_selected = false;
}

uint8_t joypad_read_impl(Jp *pad) {
	uint8_t ret = 0x0F;

	if (pad->dpad_selected) {
		for (int i = 0; i < 4; ++i) {
			int idx = (int)DPAD_BUTTONS[i];
			if (pad->buttons[idx])

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

	ret |= 0xC0;
	return ret;
}

void joypad_press_impl(Jp *pad, Button button, bool pressed) {
	pad->buttons[button] = pressed;
}

void joypad_write_impl(Jp *pad, uint8_t val) {

	uint8_t face_bit = (val >> FACE_SELECT_BIT) & 1u;

	uint8_t dpad_bit = (val >> DPAD_SELECT_BIT) & 1u;
	pad->face_selected = !face_bit;
	pad->dpad_selected = !dpad_bit;
}

void joypad_init() { joypad_init_impl(&joypad); }
uint8_t joypad_read() { return joypad_read_impl(&joypad); }
void joypad_write(uint8_t val) { joypad_write_impl(&joypad, val); }
void joypad_press(Button b, bool pressed) {
	joypad_press_impl(&joypad, b, pressed);
}
