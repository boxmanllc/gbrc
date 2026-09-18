#include "hardware/ppu.h"
#include "gbrc.h"
#include "interrupt.h"
#include <stdbool.h>
#include <stdint.h>
#include <string.h>

#define DOTS_PER_LINE 456
#define LINES_PER_FRAME 154
#define VBLANK_LINE 144
#define MODE2_DOTS 80
#define MODE3_DOTS 172

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

static Ppu ppu;

void ppu_init(void) {
	memset(&ppu, 0, sizeof(ppu));
	ppu.lcdc = 0x91;
	ppu.bgp = 0xFC;
}

static uint8_t shade(uint8_t palette, uint8_t color) {
	return (palette >> (color * 2)) & 3;
}

static void render_sprites(uint8_t ly, uint8_t *line, const uint8_t *bg_color) {
	uint8_t height = (ppu.lcdc & 0x04) ? 16 : 8;

	int chosen[10];
	int count = 0;
	for (int i = 0; i < 40 && count < 10; i++) {
		int sy = ram[0xFE00 + i * 4] - 16;
		if ((int)ly >= sy && (int)ly < sy + height) {
			chosen[count++] = i;
		}
	}

	for (int a = 0; a < count; a++) {
		for (int b = a + 1; b < count; b++) {
			int ax = ram[0xFE00 + chosen[a] * 4 + 1];
			int bx = ram[0xFE00 + chosen[b] * 4 + 1];
			bool a_lower = (ax > bx) || (ax == bx && chosen[a] > chosen[b]);
			if (!a_lower) {
				int t = chosen[a];
				chosen[a] = chosen[b];
				chosen[b] = t;
			}
		}
	}

	for (int c = 0; c < count; c++) {
		int i = chosen[c];
		int sy = ram[0xFE00 + i * 4] - 16;
		int sx = ram[0xFE00 + i * 4 + 1] - 8;
		uint8_t tile = ram[0xFE00 + i * 4 + 2];
		uint8_t attr = ram[0xFE00 + i * 4 + 3];
		bool flip_y = attr & 0x40;
		bool flip_x = attr & 0x20;
		uint8_t pal = (attr & 0x10) ? ppu.obp1 : ppu.obp0;
		bool behind_bg = attr & 0x80;

		uint8_t row = (uint8_t)((int)ly - sy);
		if (flip_y) {
			row = height - 1 - row;
		}
		if (height == 16) {
			tile &= 0xFE;
		}
		uint16_t tile_addr = 0x8000 + tile * 16 + row * 2;
		uint8_t lo = ram[tile_addr];
		uint8_t hi = ram[tile_addr + 1];

		for (int p = 0; p < 8; p++) {
			int x = sx + p;
			if (x < 0 || x >= GB_LCD_WIDTH) {
				continue;
			}
			uint8_t bit = flip_x ? p : 7 - p;
			uint8_t color =
			    (uint8_t)((((hi >> bit) & 1) << 1) | ((lo >> bit) & 1));
			if (color == 0) {
				continue;
			}
			if (behind_bg && bg_color[x] != 0) {
				continue;
			}
			line[x] = shade(pal, color);
		}
	}
}

static void render_scanline(uint8_t ly) {
	uint8_t *line = &ppu.framebuffer[ly * GB_LCD_WIDTH];
	uint8_t bg_color[GB_LCD_WIDTH];

	bool unsigned_tiles = ppu.lcdc & 0x10;
	uint16_t bg_map = (ppu.lcdc & 0x08) ? 0x9C00 : 0x9800;
	uint16_t win_map = (ppu.lcdc & 0x40) ? 0x9C00 : 0x9800;
	bool win_on_line = (ppu.lcdc & 0x20) && ppu.win_y_cond;

	for (int x = 0; x < GB_LCD_WIDTH; x++) {
		uint8_t color = 0;

		if (ppu.lcdc & 0x01) {
			bool in_window = win_on_line && (x >= (int)ppu.wx - 7);
			uint16_t map;
			uint8_t px, py;
			if (in_window) {
				map = win_map;
				px = (uint8_t)(x - ((int)ppu.wx - 7));
				py = ppu.win_line;
			} else {
				map = bg_map;
				px = (uint8_t)(x + ppu.scx);
				py = (uint8_t)(ly + ppu.scy);
			}

			uint8_t tile_id = ram[map + (py / 8) * 32 + (px / 8)];
			uint16_t tile_addr;
			if (unsigned_tiles) {
				tile_addr = 0x8000 + tile_id * 16;
			} else {
				tile_addr = 0x9000 + (int8_t)tile_id * 16;
			}

			uint8_t row = py % 8;
			uint8_t lo = ram[tile_addr + row * 2];
			uint8_t hi = ram[tile_addr + row * 2 + 1];
			uint8_t bit = 7 - (px % 8);
			color = (uint8_t)((((hi >> bit) & 1) << 1) | ((lo >> bit) & 1));
		}

		bg_color[x] = color;
		line[x] = shade(ppu.bgp, color);
	}

	if (ppu.lcdc & 0x02) {
		render_sprites(ly, line, bg_color);
	}
}

static void oam_dma(uint8_t val) {
	uint16_t src = (uint16_t)val << 8;
	for (uint16_t i = 0; i < 0xA0; i++) {
		ram[0xFE00 + i] = ram[src + i];
	}
}

static void update_mode_and_stat(void) {
	uint8_t mode;
	if (ppu.ly >= GB_LCD_HEIGHT) {
		mode = 1;
	} else if (ppu.dot < MODE2_DOTS) {
		mode = 2;
	} else if (ppu.dot < MODE2_DOTS + MODE3_DOTS) {
		mode = 3;
	} else {
		mode = 0;
	}
	ppu.mode = mode;

	bool coincidence = (ppu.ly == ppu.lyc);

	bool line = false;
	if ((ppu.stat & 0x08) && mode == 0) {
		line = true;
	}
	if ((ppu.stat & 0x10) && mode == 1) {
		line = true;
	}
	if ((ppu.stat & 0x20) && mode == 2) {
		line = true;
	}
	if ((ppu.stat & 0x40) && coincidence) {
		line = true;
	}

	if (line && !ppu.stat_line) {
		interrupt_request(INT_STAT);
	}
	ppu.stat_line = line;
}

void ppu_tick(void) {
	uint32_t now = cycles;
	uint32_t dt = (now - ppu.last_cycles) * 4;
	ppu.last_cycles = now;

	if (!(ppu.lcdc & 0x80)) {
		ppu.ly = 0;
		ppu.dot = 0;
		ppu.mode = 0;
		ppu.stat_line = false;
		return;
	}

	ppu.dot += dt;
	while (ppu.dot >= DOTS_PER_LINE) {
		ppu.dot -= DOTS_PER_LINE;

		if (ppu.ly < GB_LCD_HEIGHT) {
			if (ppu.ly == ppu.wy) {
				ppu.win_y_cond = true;
			}

			bool win_shown = ppu.win_y_cond && (ppu.lcdc & 0x20) &&
			                 ((int)ppu.wx - 7 < GB_LCD_WIDTH);

			render_scanline(ppu.ly);

			if (win_shown) {
				ppu.win_line++;
			}
		}

		ppu.ly++;
		if (ppu.ly == VBLANK_LINE) {
			interrupt_request(INT_VBLANK);
			gb_prepare_video(ppu.framebuffer);
		}
		if (ppu.ly >= LINES_PER_FRAME) {
			ppu.ly = 0;
			ppu.win_line = 0;
			ppu.win_y_cond = false;
		}

		update_mode_and_stat();
	}

	update_mode_and_stat();
}

uint8_t ppu_read(uint16_t addr) {
	ppu_tick();
	switch (addr) {
	case 0xFF40:
		return ppu.lcdc;
	case 0xFF41:
		return 0x80 | (ppu.stat & 0x78) | ((ppu.ly == ppu.lyc) ? 0x04 : 0x00) |
		       (ppu.mode & 3);
	case 0xFF42:
		return ppu.scy;
	case 0xFF43:
		return ppu.scx;
	case 0xFF44:
		return ppu.ly;
	case 0xFF45:
		return ppu.lyc;
	case 0xFF47:
		return ppu.bgp;
	case 0xFF48:
		return ppu.obp0;
	case 0xFF49:
		return ppu.obp1;
	case 0xFF4A:
		return ppu.wy;
	case 0xFF4B:
		return ppu.wx;
	default:
		return 0xFF;
	}
}

void ppu_write(uint16_t addr, uint8_t val) {
	ppu_tick();
	switch (addr) {
	case 0xFF40:
		ppu.lcdc = val;
		break;
	case 0xFF41:
		ppu.stat = val & 0x78;
		break;
	case 0xFF42:
		ppu.scy = val;
		break;
	case 0xFF43:
		ppu.scx = val;
		break;
	case 0xFF44:
		break;
	case 0xFF45:
		ppu.lyc = val;
		break;
	case 0xFF46:
		oam_dma(val);
		break;
	case 0xFF47:
		ppu.bgp = val;
		break;
	case 0xFF48:
		ppu.obp0 = val;
		break;
	case 0xFF49:
		ppu.obp1 = val;
		break;
	case 0xFF4A:
		ppu.wy = val;
		break;
	case 0xFF4B:
		ppu.wx = val;
		break;
	default:
		break;
	}
}
