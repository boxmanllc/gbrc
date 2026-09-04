#ifndef GB_RAM_H
#define GB_RAM_H
#include <stdint.h>

uint8_t read_ram(uint16_t addr);
void write_ram(uint16_t addr, uint16_t val);

#endif
