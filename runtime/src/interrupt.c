#include "interrupt.h"
#include "gb.h"
#include <stdint.h>

static uint8_t if_reg;
static uint8_t ie_reg;
uint8_t IME;

void interrupt_init() {
	if_reg = 0xE0;
	ie_reg = 0;
	IME = 0;
}

uint8_t if_read() { return if_reg | 0xE0; }
uint8_t ie_read() { return ie_reg; }
void if_write(uint8_t v) { if_reg = (v | 0xE0); }
void ie_write(uint8_t v) { ie_reg = v; }

void interrupt_request(Interrupt i) {
	if (i >= INT_UNKNOWN)
		return;
	if_reg |= (uint8_t)(1u << i);
}

uint16_t interrupt_service(void) {
	if (!IME)
		return 0xFFFF;
	uint8_t pending = if_reg & ie_reg & 0x1F;
	if (!pending)
		return 0xFFFF;

	for (int i = 0; i < 5; ++i) {
		if (pending & (1u << i)) {
			if_reg &= (uint8_t)(~(1u << i));
			IME = 0;

			sp--;
			ram[sp] = (uint8_t)(pc >> 8);
			sp--;
			ram[sp] = (uint8_t)pc;

			return (uint16_t)(0x40 + i * 8);
		}
	}
	return 0xFFFF;
}
