// Tests are ai generated

// Standalone unit check for the timer module.
// Build (from runtime/):
//   clang -Iinclude tests/timer_check.c src/timer.c src/interrupt.c \
//       -o build/timer_check
//   ./build/timer_check
//
// timer.c reads `cycles` (normally defined by the recompiled IR) and calls
// interrupt_request(), so we provide `cycles` here and link interrupt.c.

#include "gb.h"
#include "interrupt.h"
#include "timer.h"
#include <stdint.h>
#include <stdio.h>

// Machine-cycle counter the recompiled ROM would own. The timer advances off
// the delta between calls, so tests drive time by assigning to this.
uint32_t cycles;

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

// Timer register addresses.
#define DIV 0xFF04
#define TIMA 0xFF05
#define TMA 0xFF06
#define TAC 0xFF07

int main(void) {
	// --- DIV ticks once per 64 machine cycles (256 T-cycles) --------------
	cycles = 0;
	timer_init();
	cycles = 64;
	CHECK_EQ(timer_read(DIV), 1);
	cycles = 128;
	CHECK_EQ(timer_read(DIV), 2);

	// --- any write to DIV resets it ---------------------------------------
	cycles = 0;
	timer_init();
	cycles = 100;
	timer_read(DIV);       // advance the counter
	timer_write(DIV, 0xFF); // value ignored, resets to 0
	CHECK_EQ(timer_read(DIV), 0);

	// --- TIMA does not run while TAC is disabled --------------------------
	cycles = 0;
	timer_init();
	cycles = 5000;
	CHECK_EQ(timer_read(TIMA), 0);

	// --- TIMA increments at the selected rate -----------------------------
	// TAC = 0x04: enabled, freq bits 00 -> 1024 T-cycles = 256 machine cycles.
	cycles = 0;
	timer_init();
	timer_write(TAC, 0x04);
	cycles = 256;
	CHECK_EQ(timer_read(TIMA), 1);
	cycles = 512;
	CHECK_EQ(timer_read(TIMA), 2);

	// --- TAC reads back with unused upper bits set ------------------------
	cycles = 0;
	timer_init();
	timer_write(TAC, 0x05);
	CHECK_EQ(timer_read(TAC), 0xFD); // 0x05 | 0xF8

	// --- TIMA overflow reloads TMA and requests INT_TIMER -----------------
	cycles = 0;
	timer_init();
	interrupt_init();
	timer_write(TMA, 0xAB);  // reload value
	timer_write(TIMA, 0xFF); // one tick from overflow
	timer_write(TAC, 0x05);  // enable, freq 01 -> 16 T-cycles = 4 machine cyc.
	cycles = 4;              // exactly one TIMA tick -> overflow
	CHECK_EQ(timer_read(TIMA), 0xAB);
	CHECK_EQ(if_read() & (1 << INT_TIMER), (1 << INT_TIMER));

	// --- unmapped register reads as 0xFF ----------------------------------
	cycles = 0;
	timer_init();
	CHECK_EQ(timer_read(0xFF10), 0xFF);

	if (failures == 0) {
		puts("timer_check: all passed");
		return 0;
	}
	printf("timer_check: %d failure(s)\n", failures);
	return 1;
}
