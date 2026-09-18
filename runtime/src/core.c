#include "gbrc.h"
#include "hardware/apu.h"
#include "hardware/joypad.h"
#include "hardware/ppu.h"
#include "hardware/timer.h"
#include "interrupt.h"
#include "profile.h"
#include <stdint.h>
#include <stdio.h>

static const char *g_profile_path;

static const char *mbc_type_name(uint8_t t) {
	switch (t) {
	case 0x00:
		return "ROM ONLY";
	case 0x01:
		return "MBC1";
	case 0x02:
		return "MBC1+RAM";
	case 0x03:
		return "MBC1+RAM+BATTERY";
	case 0x05:
		return "MBC2";
	case 0x06:
		return "MBC2+BATTERY";
	default:
		return "UNKNOWN";
	}
}

static int ram_size_bytes(uint8_t t) {
	switch (t) {
	case 0x02:
		return 8 * 1024;
	case 0x03:
		return 32 * 1024;
	case 0x04:
		return 128 * 1024;
	case 0x05:
		return 64 * 1024;
	default:
		return 0;
	}
}

static void rom_title(char *out, size_t n) {
	size_t j = 0;
	for (size_t i = 0; i < 16 && j + 1 < n; i++) {
		uint8_t c = ram[0x0134 + i];
		if (c == 0) {
			break;
		}
		if (c >= 0x20 && c < 0x7F) {
			out[j++] = (char)c;
		}
	}
	while (j > 0 && out[j - 1] == ' ') {
		j--;
	}
	out[j] = '\0';
}

static void print_rom_info(void) {
	char title[32];
	rom_title(title, sizeof(title));

	int rom_size_byte = ram[0x0148];
	if (rom_size_byte > 8) {
		rom_size_byte = 8;
	}

	printf("ROM Information:\n");
	printf("  Title: %s\n", title);
	printf("  MBC Type: %s\n", mbc_type_name(ram[0x0147]));
	printf("  ROM Size: %.2f KiB\n",
	       (double)(32 * 1024 * (1 << rom_size_byte)) / 1024.0);
	printf("  RAM Size: %.2f KiB\n",
	       (double)ram_size_bytes(ram[0x0149]) / 1024.0);
	printf("  CGB Support: %s\n", (ram[0x0143] & 0x80) ? "true" : "false");
	printf("  ROM Version: %d\n", ram[0x014C]);
}

bool gb_attach(const char *title_override) {
	rom_init();
	print_rom_info();

	char title[32];
	rom_title(title, sizeof(title));

	const char *window_title = title_override;
	if (!window_title || window_title[0] == '\0') {
		window_title = title[0] ? title : "gbrc";
	}

	interrupt_init();
	timer_init();
	joypad_init();
	ppu_init();
	apu_init();

	g_budget = GB_CYCLES_PER_FRAME;

	return gb_init(window_title);
}

void gb_set_profile_path(const char *path) { g_profile_path = path; }

void gb_profile_tick(uint32_t frame) {
	if (g_profile_path && frame % 60 == 0) {
		profile_dump(g_profile_path);
	}
}

void gb_profile_flush(void) { profile_dump(g_profile_path); }
