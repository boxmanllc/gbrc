#ifndef GB_PPU_H
#define GB_PPU_H

#include <stdint.h>

void ppu_init(void);
void ppu_tick(void);
uint8_t ppu_read(uint16_t addr);
void ppu_write(uint16_t addr, uint8_t val);

#endif
