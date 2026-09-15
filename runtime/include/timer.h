#ifndef GB_TIMER_H
#define GB_TIMER_H
#include <stdint.h>

typedef struct {
	uint32_t tima_accum;
	uint32_t last_cycles;
	uint16_t div_counter;
	uint8_t tima;
	uint8_t tma;
	uint8_t tac;
} Timer;

void timer_init_impl(Timer *tmr);
void timer_tick_impl(Timer *tmr);
uint8_t timer_read_impl(Timer *tmr, uint16_t addr);
void timer_write_impl(Timer *tmr, uint16_t addr, uint8_t val);

void timer_init(void);
void timer_tick(void);
uint8_t timer_read(uint16_t addr);
void timer_write(uint16_t addr, uint8_t val);

#endif
