#include "hardware/joypad.h"
#include "interrupt.h"
#include <stdbool.h>
#include <stdint.h>
#include <string.h>

#define FACE_SELECT_BIT 5
#define DPAD_SELECT_BIT 4

typedef struct {
	bool buttons[8];
	bool dpad_selected;
	bool face_selected;
} Jp;

static Jp joypad;

static const Button dpad_buttons[4] = {RIGHT, LEFT, UP, DOWN};
static const Button face_buttons[4] = {A, B, SELECT, START};

void joypad_init(void) {
	memset(joypad.buttons, false, sizeof(joypad.buttons));
	joypad.dpad_selected = false;
	joypad.face_selected = false;
}

uint8_t joypad_read(void) {
	uint8_t ret = 0x0F;

	if (joypad.dpad_selected) {
		for (int i = 0; i < 4; i++) {
			int idx = (int)dpad_buttons[i];
			if (joypad.buttons[idx]) {
				ret &= (uint8_t)~(1u << (idx & 3));
			}
		}
	}

	if (joypad.face_selected) {
		for (int i = 0; i < 4; i++) {
			int idx = (int)face_buttons[i];
			if (joypad.buttons[idx]) {
				ret &= (uint8_t)~(1u << (idx & 3));
			}
		}
	}

	return ret | 0xC0;
}

void joypad_write(uint8_t val) {
	joypad.face_selected = !((val >> FACE_SELECT_BIT) & 1u);
	joypad.dpad_selected = !((val >> DPAD_SELECT_BIT) & 1u);
}

void joypad_press(Button button, bool pressed) {
	int idx = (int)button;
	if (idx < 0 || idx > 7) {
		return;
	}

	bool selected =
	    (idx >= (int)RIGHT) ? joypad.dpad_selected : joypad.face_selected;
	if (pressed && !joypad.buttons[idx] && selected) {
		interrupt_request(INT_JOYPAD);
	}

	joypad.buttons[idx] = pressed;
}
