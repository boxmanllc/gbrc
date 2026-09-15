#ifndef GB_PPU_H
#define GB_PPU_H
#include <stdbool.h>
#include <stdint.h>

#define GB_LCD_WIDTH 160
#define GB_LCD_HEIGHT 144

typedef struct {
	// registers $FF40-$FF4B. DMA ($FF46) acts on write and isn't stored here.
	uint8_t lcdc; // $FF40 control
	uint8_t stat; // $FF41 status (only the writable select bits are kept here)
	uint8_t scy;  // $FF42 bg scroll y
	uint8_t scx;  // $FF43 bg scroll x
	uint8_t ly;   // $FF44 current scanline (read-only to the cpu)
	uint8_t lyc;  // $FF45 ly compare
	uint8_t bgp;  // $FF47 bg palette
	uint8_t obp0; // $FF48 obj palette 0
	uint8_t obp1; // $FF49 obj palette 1
	uint8_t wy;   // $FF4A window y
	uint8_t wx;   // $FF4B window x (+ 7)

	uint32_t last_cycles; // cpu cycle count at the last catch-up
	uint32_t dot;         // dot position within the current scanline (0-455)
	uint8_t mode;         // current mode: 2 oam, 3 draw, 0 hblank, 1 vblank
	bool stat_line;       // previous stat irq line, for rising-edge detection

	// output: one byte per pixel, a 2-bit shade (0 lightest .. 3 darkest)
	uint8_t framebuffer[GB_LCD_HEIGHT * GB_LCD_WIDTH];
} Ppu;

void ppu_init_impl(Ppu *ppu);
void ppu_tick_impl(Ppu *ppu);
uint8_t ppu_read_impl(Ppu *ppu, uint16_t addr);
void ppu_write_impl(Ppu *ppu, uint16_t addr, uint8_t val);

// no-arg wrappers over the one ppu
void ppu_init(void);
void ppu_tick(void);
uint8_t ppu_read(uint16_t addr);
void ppu_write(uint16_t addr, uint8_t val);

// Pluggable rendering backend (SDL2, web, ...): called once per completed
// frame with the 160x144 framebuffer. NULL until a backend registers one.
extern void (*ppu_present)(const uint8_t *framebuffer);

#endif
