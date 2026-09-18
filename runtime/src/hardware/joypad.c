#include "hardware/joypad.h"
#include "gbrc.h"
#include "interrupt.h"
#include <stdbool.h>
#include <stdint.h>
#include <string.h>

#define FACE_SELECT_BIT 5
#define DPAD_SELECT_BIT 4

typedef struct {
	bool buttons[GB_BUTTON_COUNT];
	bool dpad_selected;
	bool face_selected;
} Joypad;

static Joypad joypad;

static const gb_button DPAD_BUTTONS[4] = {
    GB_BUTTON_RIGHT,
    GB_BUTTON_LEFT,
    GB_BUTTON_UP,
    GB_BUTTON_DOWN,
};
static const gb_button FACE_BUTTONS[4] = {
    GB_BUTTON_A,
    GB_BUTTON_B,
    GB_BUTTON_SELECT,
    GB_BUTTON_START,
};

void joypad_init(void) { memset(&joypad, 0, sizeof(joypad)); }

uint8_t joypad_read(void) {
	uint8_t ret = 0x0F;

	if (joypad.dpad_selected) {
		for (int i = 0; i < 4; i++) {
			int idx = (int)DPAD_BUTTONS[i];
			if (joypad.buttons[idx]) {
				ret &= (uint8_t)~(1u << (idx & 3));
			}
		}
	}

	if (joypad.face_selected) {
		for (int i = 0; i < 4; i++) {
			int idx = (int)FACE_BUTTONS[i];
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

void gb_set_button(gb_button button, bool pressed) {
	int idx = (int)button;
	if (idx < 0 || idx >= GB_BUTTON_COUNT) {
		return;
	}

	bool selected = (idx >= (int)GB_BUTTON_RIGHT) ? joypad.dpad_selected
	                                              : joypad.face_selected;
	if (pressed && !joypad.buttons[idx] && selected) {
		interrupt_request(INT_JOYPAD);
	}

	joypad.buttons[idx] = pressed;
}
