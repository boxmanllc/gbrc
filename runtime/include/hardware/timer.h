#ifndef GB_TIMER_H
#define GB_TIMER_H

#include <stdint.h>

void timer_init(void);
void timer_tick(void);
uint8_t timer_read(uint16_t addr);
void timer_write(uint16_t addr, uint8_t val);

#endif
