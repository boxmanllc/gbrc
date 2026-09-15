// Tests are ai generated

// Standalone unit check for the ppu module.
// Build (from runtime/):
//   clang -Iinclude tests/ppu_check.c src/ppu.c src/interrupt.c \
//       -o build/ppu_check
//   ./build/ppu_check
//
// ppu.c reads ram[] and `cycles` (normally the recompiled IR defines them) and
// calls interrupt_request(), so we provide ram/cycles here and link interrupt.c.

#include "gb.h"
#include "interrupt.h"
#include "ppu.h"
#include <stdint.h>
#include <stdio.h>
#include <string.h>

// State the recompiled ROM would own.
uint8_t ram[0x10000];
uint32_t cycles;

static int failures = 0;

#define CHECK_EQ(actual, expected)                                             \
	do {                                                                       \
		unsigned long _a = (unsigned long)(actual);                            \
		unsigned long _e = (unsigned long)(expected);                          \
		if (_a != _e) {                                                        \
			printf("FAIL line %d: %s = %lu (0x%lX), expected %lu (0x%lX)\n",   \
			       __LINE__, #actual, _a, _a, _e, _e);                         \
			failures++;                                                        \
		}                                                                      \
	} while (0)

// Backend hook capture.
static const uint8_t *captured_fb;
static int present_count;
static void on_present(const uint8_t *fb) {
	captured_fb = fb;
	present_count++;
}

#define FRAME_CYCLES 16416 // machine cycles from line 0 to line 144 (VBlank)
#define LINE_CYCLES 114    // 456 dots / 4

// fill tile `id` in the $8000 block with a solid color (both bitplanes set/clear)
static void fill_tile(uint16_t base, uint8_t id, uint8_t color) {
	uint8_t lo = (color & 1) ? 0xFF : 0x00;
	uint8_t hi = (color & 2) ? 0xFF : 0x00;
	for (int r = 0; r < 8; r++) {
		ram[base + id * 16 + r * 2] = lo;
		ram[base + id * 16 + r * 2 + 1] = hi;
	}
}

int main(void) {
	// --- background renders a known tile; VBlank fires; frame presented -----
	memset(ram, 0, sizeof(ram));
	cycles = 0;
	interrupt_init();
	ppu_init();
	ppu_present = on_present;
	present_count = 0;
	captured_fb = 0;

	fill_tile(0x8000, 0, 3); // tile 0 = solid color 3
	// bg map ($9800) is all zeros -> tile 0 everywhere
	ppu_write(0xFF47, 0xE4); // BGP identity: 3->3

	cycles = FRAME_CYCLES;
	ppu_tick();

	CHECK_EQ(present_count, 1);
	CHECK_EQ(captured_fb != 0, 1);
	if (captured_fb) {
		CHECK_EQ(captured_fb[0], 3);              // top-left
		CHECK_EQ(captured_fb[100 * 160 + 50], 3); // somewhere in the middle
	}
	CHECK_EQ(if_read() & 0x01, 0x01); // VBlank interrupt requested

	// --- LY advances with elapsed cycles ----------------------------------
	memset(ram, 0, sizeof(ram));
	cycles = 0;
	ppu_init();
	cycles = LINE_CYCLES; // exactly one scanline
	CHECK_EQ(ppu_read(0xFF44), 1);
	cycles = 5 * LINE_CYCLES;
	CHECK_EQ(ppu_read(0xFF44), 5);

	// --- LCD off holds LY at 0 --------------------------------------------
	cycles = 0;
	ppu_init();
	ppu_write(0xFF40, 0x00); // LCD off
	cycles = 1000;
	CHECK_EQ(ppu_read(0xFF44), 0);

	// --- LYC coincidence sets STAT bit 2 ----------------------------------
	cycles = 0;
	ppu_init();
	ppu_write(0xFF45, 5); // LYC = 5
	cycles = 5 * LINE_CYCLES;
	CHECK_EQ((ppu_read(0xFF41) >> 2) & 1, 1);
	CHECK_EQ(ppu_read(0xFF44), 5);

	// --- OAM DMA copies 160 bytes into $FE00 ------------------------------
	memset(ram, 0, sizeof(ram));
	cycles = 0;
	ppu_init();
	for (int i = 0; i < 0xA0; i++)
		ram[0xC000 + i] = (uint8_t)i;
	ppu_write(0xFF46, 0xC0); // DMA from $C000
	CHECK_EQ(ram[0xFE00], 0x00);
	CHECK_EQ(ram[0xFE00 + 0x9F], 0x9F);

	// --- a sprite draws over the background --------------------------------
	memset(ram, 0, sizeof(ram));
	cycles = 0;
	interrupt_init();
	ppu_init();
	ppu_present = on_present;
	present_count = 0;
	captured_fb = 0;

	fill_tile(0x8000, 0, 0); // bg tile 0 = color 0 (transparent-ish, shade 0)
	fill_tile(0x8000, 1, 3); // sprite tile 1 = solid color 3
	memset(&ram[0xFE00], 0, 0xA0); // clear OAM
	ram[0xFE00] = 16;              // sprite 0 y -> screen row 0
	ram[0xFE01] = 8;               // x -> screen col 0
	ram[0xFE02] = 1;               // tile 1
	ram[0xFE03] = 0;               // attrs
	ppu_write(0xFF40, 0x93);       // enable obj (bit1) on top of boot lcdc
	ppu_write(0xFF47, 0xE4);       // BGP: color0 -> shade 0
	ppu_write(0xFF48, 0xE4);       // OBP0: color3 -> shade 3

	cycles = FRAME_CYCLES;
	ppu_tick();
	if (captured_fb) {
		CHECK_EQ(captured_fb[0], 3);  // sprite pixel
		CHECK_EQ(captured_fb[50], 0); // bg pixel outside the sprite
	}

	if (failures == 0) {
		puts("ppu_check: all passed");
		return 0;
	}
	printf("ppu_check: %d failure(s)\n", failures);
	return 1;
}
