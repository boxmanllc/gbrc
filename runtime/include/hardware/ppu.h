#ifndef GB_PPU_H
#define GB_PPU_H

#include <stdint.h>

#define GB_LCD_WIDTH 160
#define GB_LCD_HEIGHT 144

void ppu_init(void);
void ppu_tick(void);
uint8_t ppu_read(uint16_t addr);
void ppu_write(uint16_t addr, uint8_t val);

extern void (*ppu_present)(const uint8_t *framebuffer);

#endif
