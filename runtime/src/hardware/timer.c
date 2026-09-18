#include "hardware/timer.h"
#include "gbrc.h"
#include "interrupt.h"
#include <stdint.h>
#include <string.h>

#define DIV 0xFF04
#define TIMA 0xFF05
#define TMA 0xFF06
#define TAC 0xFF07

typedef struct {
	uint32_t tima_accum;
	uint32_t last_cycles;
	uint16_t div_counter;
	uint8_t tima;
	uint8_t tma;
	uint8_t tac;
} Timer;

static Timer tmr;

static const uint32_t tima_period[4] = {1024, 16, 64, 256};

void timer_init(void) { memset(&tmr, 0, sizeof(tmr)); }

void timer_tick(void) {
	uint32_t now = cycles;
	uint32_t dt = (now - tmr.last_cycles) * 4;
	tmr.last_cycles = now;
	tmr.div_counter += dt;

	if (!(tmr.tac & 0x04)) {
		return;
	}

	tmr.tima_accum += dt;
	uint32_t period = tima_period[tmr.tac & 3];
	while (tmr.tima_accum >= period) {
		tmr.tima_accum -= period;
		if (tmr.tima == 0xFF) {
			tmr.tima = tmr.tma;
			interrupt_request(INT_TIMER);
		} else {
			tmr.tima++;
		}
	}
}

uint8_t timer_read(uint16_t addr) {
	timer_tick();
	switch (addr) {
	case DIV:
		return (uint8_t)(tmr.div_counter >> 8);
	case TIMA:
		return tmr.tima;
	case TMA:
		return tmr.tma;
	case TAC:
		return tmr.tac | 0xF8;
	default:
		return 0xFF;
	}
}

void timer_write(uint16_t addr, uint8_t val) {
	timer_tick();
	switch (addr) {
	case DIV:
		tmr.div_counter = 0;
		break;
	case TIMA:
		tmr.tima = val;
		break;
	case TMA:
		tmr.tma = val;
		break;
	case TAC:
		tmr.tac = val & 7;
		break;
	default:
		break;
	}
}
