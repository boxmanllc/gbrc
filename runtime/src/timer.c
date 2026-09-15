#include "timer.h"
#include "gb.h"
#include "interrupt.h"
#include <stdint.h>

#define DIV 0xFF04
#define TIMA 0xFF05
#define TMA 0xFF06
#define TAC 0xFF07

static Timer tmr;

static const uint32_t tima_period[4] = {1024, 16, 64, 256};

void timer_init_impl(Timer *tmr) {
	tmr->div_counter = 0;
	tmr->tima = tmr->tma = tmr->tac = 0;
	tmr->tima_accum = 0;
	tmr->last_cycles = 0;
}

void timer_tick_impl(Timer *tmr) {

	uint32_t now = cycles;

	uint32_t dt = (now - tmr->last_cycles) * 4;
	tmr->last_cycles = now;

	tmr->div_counter += dt;

	if (tmr->tac & 0x04) {
		tmr->tima_accum += dt;

		uint32_t period = tima_period[tmr->tac & 3];
		while (tmr->tima_accum >= period) {
			tmr->tima_accum -= period;

			if (tmr->tima == 0xFF) {
				tmr->tima = tmr->tma;
				interrupt_request(INT_TIMER);
			} else {
				tmr->tima++;
			}
		}
	}
}

uint8_t timer_read_impl(Timer *tmr, uint16_t addr) {
	timer_tick_impl(tmr);
	switch (addr) {
	case DIV:
		return (uint8_t)(tmr->div_counter >> 8);
	case TIMA:
		return tmr->tima;
	case TMA:
		return tmr->tma;
	case TAC:

		return tmr->tac | 0xF8;
	}
	return 0xFF;
}

void timer_write_impl(Timer *tmr, uint16_t addr, uint8_t val) {
	timer_tick_impl(tmr);
	switch (addr) {
	case DIV:
		tmr->div_counter = 0;
		break;
	case TIMA:
		tmr->tima = val;
		break;
	case TMA:
		tmr->tma = val;
		break;
	case TAC:
		tmr->tac = val & 7;
		break;
	}
}

void timer_init(void) { timer_init_impl(&tmr); };
void timer_tick(void) { timer_tick_impl(&tmr); };
uint8_t timer_read(uint16_t addr) { return timer_read_impl(&tmr, addr); };
void timer_write(uint16_t addr, uint8_t val) {
	timer_write_impl(&tmr, addr, val);
};
