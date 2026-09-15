// Tests are ai generated

// Standalone unit check for the joypad module.
// Build (from runtime/):
//   clang -Iinclude tests/joypad_check.c src/joypad.c -o build/joypad_check
//   ./build/joypad_check
// joypad.c depends on nothing else, so this links on its own.

#include "joypad.h"
#include <stdint.h>
#include <stdio.h>

static int failures = 0;

#define CHECK_EQ(actual, expected)                                             \
	do {                                                                       \
		unsigned long _a = (unsigned long)(actual);                            \
		unsigned long _e = (unsigned long)(expected);                          \
		if (_a != _e) {                                                        \
			printf("FAIL line %d: %s = %lu (0x%lX), expected %lu (0x%lX)\n",   \
			       __LINE__, #actual, _a, _a, _e, _e);                         \
			failures++;                                                        \
		}                                                                      \
	} while (0)

// Select-register writes (active-low): P14/P15 low = that group selected.
#define SELECT_DPAD 0x20 // bit4 = 0 -> d-pad,   bit5 = 1
#define SELECT_FACE 0x10 // bit5 = 0 -> buttons, bit4 = 1

int main(void) {
	Jp pad;

	// --- upper bits 6,7 always read 1 -------------------------------------
	joypad_init_impl(&pad);
	CHECK_EQ(joypad_read_impl(&pad) & 0xC0, 0xC0);

	// --- nothing pressed: active-low -> input nibble all 1s ---------------
	joypad_init_impl(&pad);
	joypad_write_impl(&pad, SELECT_DPAD);
	CHECK_EQ(joypad_read_impl(&pad) & 0x0F, 0x0F);

	// --- d-pad selected: RIGHT is bit0, DOWN is bit3 (pressed = 0) --------
	joypad_init_impl(&pad);
	joypad_write_impl(&pad, SELECT_DPAD);
	joypad_press_impl(&pad, RIGHT, true);
	CHECK_EQ(joypad_read_impl(&pad) & 0x0F, 0x0E);
	joypad_press_impl(&pad, DOWN, true);
	CHECK_EQ(joypad_read_impl(&pad) & 0x0F, 0x06);

	// --- group isolation: a face button must not show while d-pad chosen --
	joypad_init_impl(&pad);
	joypad_write_impl(&pad, SELECT_DPAD);
	joypad_press_impl(&pad, A, true); // A is bit0 of the FACE group
	CHECK_EQ(joypad_read_impl(&pad) & 0x0F, 0x0F);

	// --- face selected: A is bit0, START is bit3 --------------------------
	joypad_init_impl(&pad);
	joypad_write_impl(&pad, SELECT_FACE);
	joypad_press_impl(&pad, A, true);
	CHECK_EQ(joypad_read_impl(&pad) & 0x0F, 0x0E);
	joypad_init_impl(&pad);
	joypad_write_impl(&pad, SELECT_FACE);
	joypad_press_impl(&pad, START, true);
	CHECK_EQ(joypad_read_impl(&pad) & 0x0F, 0x07);

	// --- singleton wrappers (what ram.c actually calls) -------------------
	joypad_init();
	joypad_write(SELECT_DPAD);
	joypad_press(RIGHT, true);
	CHECK_EQ(joypad_read() & 0x0F, 0x0E);
	joypad_press(RIGHT, false);
	CHECK_EQ(joypad_read() & 0x0F, 0x0F);

	if (failures == 0) {
		puts("joypad_check: all passed");
		return 0;
	}
	printf("joypad_check: %d failure(s)\n", failures);
	return 1;
}
