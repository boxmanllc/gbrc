#ifndef GB_JOYPAD_H
#define GB_JOYPAD_H

#include <stdint.h>

void joypad_init(void);
uint8_t joypad_read(void);
void joypad_write(uint8_t val);

#endif
