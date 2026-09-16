#ifndef GB_JOYPAD_H
#define GB_JOYPAD_H

#include <stdbool.h>
#include <stdint.h>

typedef enum {
	DOWN = 7,
	UP = 6,
	LEFT = 5,
	RIGHT = 4,
	START = 3,
	SELECT = 2,
	B = 1,
	A = 0,
} Button;

void joypad_init(void);
uint8_t joypad_read(void);
void joypad_write(uint8_t val);
void joypad_press(Button b, bool pressed);

#endif
