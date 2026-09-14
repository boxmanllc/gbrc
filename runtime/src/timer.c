#include "timer.h"
#include "gb.h"
#include "interrupt.h"
#include <stdint.h>

static Timer tmr;

static const uint32_t tima_period[4] = {1024, 16, 64, 256};

void timer_init_impl(Timer *tmr) {
	tmr->div_counter = 0;
	tmr->tima = tmr->tma = tmr->tac = 0;
	tmr->tima_accum = 0;
	tmr->last_cycles = 0;
}

void timer_tick_impl(Timer *tmr) {
	// no. of M-cycles currently
	uint32_t now = cycles;
	// no. of T-cycles since last catch up
	uint32_t dt = (now - tmr->last_cycles) * 4;
	tmr->last_cycles = now;

	tmr->div_counter += dt;

	// bit 2 of TAC checks if timer enabled
	// if enabled, run TIMA
	if (tmr->tac & 0x04) {
		tmr->tima_accum += dt;
		/*
		 * period is how many T-cycles make one TIMA increment
		 * only get lower 2 bytes (TAC & 3)
		 * tima_period[0] => 0b00 : CPU Clock / 1024
		 * tima_period[1] => 0b01 : CPU Clock / 16
		 * tima_period[2] => 0b10 : CPU Clock / 64
		 * tima_period[3] => 0b11 : CPU Clock / 256
		 */
		uint32_t period = tima_period[tmr->tac & 3];
		while (tmr->tima_accum >= period) {
			tmr->tima_accum -= period;
			// if overflow
			if (tmr->tima == 0xFF) {
				tmr->tima = tmr->tma; // reload from TMA
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
	case 0xFF04:
		return (uint8_t)(tmr->div_counter >> 8);
	case 0xFF05:
		return tmr->tima;
	case 0xFF06:
		return tmr->tma;
	case 0xFF07:
		// unused top 5 bits read as 1
		// 0xF8 = 0b1111_1000
		return tmr->tac | 0xF8;
	}
	return 0xFF;
}

void timer_write_impl(Timer *tmr, uint16_t addr, uint8_t val) {
	timer_tick_impl(tmr);
	switch (addr) {
	case 0xFF04:
		tmr->div_counter = 0;
		break;
	case 0xFF05:
		tmr->tima = val;
		break;
	case 0xFF06:
		tmr->tma = val;
		break;
	case 0xFF07:
		tmr->tac = val & 7;
		break;
	}
}

// no arg wrappers
void timer_init(void) { timer_init_impl(&tmr); };
void timer_tick(void) { timer_tick_impl(&tmr); };
uint8_t timer_read(uint16_t addr) { return timer_read_impl(&tmr, addr); };
void timer_write(uint16_t addr, uint8_t val) {
	timer_write_impl(&tmr, addr, val);
};
