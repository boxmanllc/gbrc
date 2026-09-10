// Tests are ai generated

// Standalone unit check for the interrupt module.
// Build (from runtime/):
//   clang -Iinclude tests/interrupt_check.c src/interrupt.c -o
//   build/interrupt_check ./build/interrupt_check
// interrupt.c depends on nothing else, so this links on its own.

#include "interrupt.h"
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

// Dummy handlers record that they fired (the recompiled ROM would install real
// ones). void(void) shape matches int_handlers[].
static int vblank_calls, timer_calls, joypad_calls;
static void vblank_handler(void) { vblank_calls++; }
static void timer_handler(void) { timer_calls++; }
static void joypad_handler(void) { joypad_calls++; }

static void reset(void) {
	interrupt_init();
	for (int i = 0; i < 5; ++i)
		int_handlers[i] = 0;
	vblank_calls = timer_calls = joypad_calls = 0;
}

int main(void) {
	// --- init state -------------------------------------------------------
	reset();
	CHECK_EQ(if_read(), 0xE0); // unused top bits read as 1
	CHECK_EQ(ie_read(), 0x00);
	CHECK_EQ(IME, 0);

	// --- request sets the IF bit; top bits stay high ----------------------
	reset();
	interrupt_request(INT_TIMER); // bit 2
	CHECK_EQ(if_read(), 0xE4);    // 0xE0 | (1<<2)

	// --- IE round-trips ---------------------------------------------------
	reset();
	ie_write(0x1F);
	CHECK_EQ(ie_read(), 0x1F);

	// --- gating: IME == 0 -> nothing serviced, request stays pending ------
	reset();
	ie_write(1 << INT_TIMER);
	interrupt_request(INT_TIMER);
	int_handlers[INT_TIMER] = timer_handler;
	IME = 0;
	interrupt_dispatch();
	CHECK_EQ(timer_calls, 0);
	CHECK_EQ(if_read() & (1 << INT_TIMER), (1 << INT_TIMER)); // still pending

	// --- gating: enabled bit clear in IE -> not serviced ------------------
	reset();
	ie_write(0);
	interrupt_request(INT_TIMER);
	int_handlers[INT_TIMER] = timer_handler;
	IME = 1;
	interrupt_dispatch();
	CHECK_EQ(timer_calls, 0);

	// --- happy path: IME=1 and enabled -> handler runs, IF+IME cleared ----
	reset();
	ie_write(1 << INT_TIMER);
	interrupt_request(INT_TIMER);
	int_handlers[INT_TIMER] = timer_handler;
	IME = 1;
	interrupt_dispatch();
	CHECK_EQ(timer_calls, 1);
	CHECK_EQ(if_read() & (1 << INT_TIMER), 0); // acknowledged
	CHECK_EQ(IME, 0);                          // master disabled on entry

	// --- priority: lowest bit wins, only one serviced per dispatch --------
	reset();
	ie_write((1 << INT_VBLANK) | (1 << INT_JOYPAD));
	interrupt_request(INT_VBLANK);
	interrupt_request(INT_JOYPAD);
	int_handlers[INT_VBLANK] = vblank_handler;
	int_handlers[INT_JOYPAD] = joypad_handler;
	IME = 1;
	interrupt_dispatch();
	CHECK_EQ(vblank_calls, 1); // VBlank (bit 0) beats Joypad (bit 4)
	CHECK_EQ(joypad_calls, 0);
	CHECK_EQ(if_read() & (1 << INT_JOYPAD), (1 << INT_JOYPAD)); // still pending
	IME = 1; // handler re-enabled it
	interrupt_dispatch();
	CHECK_EQ(joypad_calls, 1); // now serviced
	CHECK_EQ(IME, 0);

	// --- NULL handler is safe: still acknowledges (clears IF + IME) -------
	reset();
	ie_write(1 << INT_STAT);
	interrupt_request(INT_STAT);
	// no handler installed -> int_handlers[INT_STAT] == NULL
	IME = 1;
	interrupt_dispatch();
	CHECK_EQ(if_read() & (1 << INT_STAT), 0);
	CHECK_EQ(IME, 0);

	if (failures == 0) {
		puts("interrupt_check: all passed");
		return 0;
	}
	printf("interrupt_check: %d failure(s)\n", failures);
	return 1;
}
