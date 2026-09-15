#include "interrupt.h"
#include <stdint.h>

static uint8_t if_reg;
static uint8_t ie_reg;
uint8_t IME;
void (*int_handlers[5])(void);

void interrupt_init() {
	if_reg = 0xE0;
	ie_reg = 0;
	IME = 0;
}

// IF: bits 7,6,5 are useless. 0xE0 = 0b1110_0000
uint8_t if_read() { return if_reg | 0xE0; }
uint8_t ie_read() { return ie_reg; }
void if_write(uint8_t v) { if_reg = (v | 0xE0); }
void ie_write(uint8_t v) { ie_reg = v; }

void interrupt_request(Interrupt i) {
	if (i >= INT_UNKNOWN)
		return;
	if_reg |= (uint8_t)(1u << i);
}

void interrupt_dispatch() {
	if (!IME)
		return;
	// 0x1F = 0b0001_1111
	uint8_t pending = if_reg & ie_reg & 0x1F;
	if (!pending)
		return;

	// we only need lower 5 bits
	for (int i = 0; i < 5; ++i) {
		if (pending & (1u << i)) {
			/*
			 * if_reg = 0b0000_1100 (serial and timer pending)
			 *      &   0b1111_0111 (eg. for i = 4)
			 *          -----------
			 *          0b0000_0100 (serial cleared, timer still pending)
			 */
			if_reg &= (uint8_t)(~(1u << i));
			IME = 0;

			if (int_handlers[i])
				int_handlers[i]();
			return;
		}
	}
}
