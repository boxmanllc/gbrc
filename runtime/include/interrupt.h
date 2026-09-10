#ifndef GB_INTERRUPT_H
#define GB_INTERRUPT_H
#include <stdint.h>

typedef enum {
	INT_VBLANK = 0,
	INT_STAT = 1,
	INT_TIMER = 2,
	INT_SERIAL = 3,
	INT_JOYPAD = 4,
} Interrupt;

extern uint8_t IME;
extern void (*int_handlers[5])(void);

uint8_t if_read();
uint8_t ie_read();
void if_write(uint8_t v);
void ie_write(uint8_t v);

void interrupt_request(Interrupt i);
void interrupt_dispatch();
void interrupt_init();

#endif
