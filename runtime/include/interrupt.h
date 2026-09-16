#ifndef GB_INTERRUPT_H
#define GB_INTERRUPT_H

#include <stdint.h>

typedef enum {
	INT_VBLANK = 0,
	INT_STAT = 1,
	INT_TIMER = 2,
	INT_SERIAL = 3,
	INT_JOYPAD = 4,
	INT_UNKNOWN = 5,
} Interrupt;

extern uint8_t IME;

void interrupt_init(void);

uint8_t if_read(void);
uint8_t ie_read(void);
void if_write(uint8_t v);
void ie_write(uint8_t v);

void interrupt_request(Interrupt i);
uint16_t interrupt_service(void);

#endif
