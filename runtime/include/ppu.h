#ifndef GB_PPU_H
#define GB_PPU_H
#include <stdbool.h>
#include <stdint.h>

#define GB_LCD_WIDTH 160
#define GB_LCD_HEIGHT 144

typedef struct {

	uint8_t lcdc;
	uint8_t stat;
	uint8_t scy;
	uint8_t scx;
	uint8_t ly;
	uint8_t lyc;
	uint8_t bgp;
	uint8_t obp0;
	uint8_t obp1;
	uint8_t wy;
	uint8_t wx;

	uint32_t last_cycles;
	uint32_t dot;
	uint8_t mode;
	bool stat_line;

	uint8_t win_line;
	bool win_y_cond;

	uint8_t framebuffer[GB_LCD_HEIGHT * GB_LCD_WIDTH];
} Ppu;

void ppu_init_impl(Ppu *ppu);
void ppu_tick_impl(Ppu *ppu);
uint8_t ppu_read_impl(Ppu *ppu, uint16_t addr);
void ppu_write_impl(Ppu *ppu, uint16_t addr, uint8_t val);

void ppu_init(void);
void ppu_tick(void);
uint8_t ppu_read(uint16_t addr);
void ppu_write(uint16_t addr, uint8_t val);

extern void (*ppu_present)(const uint8_t *framebuffer);

#endif
