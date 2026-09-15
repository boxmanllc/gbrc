#include "apu.h"
#include "frontend.h"
#include "gb.h"
#include "interrupt.h"
#include "joypad.h"
#include "ppu.h"
#include "profile.h"
#include "timer.h"
#include <stdint.h>
#include <stdio.h>
#include <string.h>

#define CYCLES_PER_FRAME 17556

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
		if (c == 0)
			break;
		if (c >= 0x20 && c < 0x7F)
			out[j++] = (char)c;
	}
	while (j > 0 && out[j - 1] == ' ')
		j--;
	out[j] = '\0';
}

static void print_rom_info(void) {
	char title[32];
	rom_title(title, sizeof(title));

	int rom_size_byte = ram[0x0148];
	if (rom_size_byte > 8)
		rom_size_byte = 8;

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

void gb_init() {
	interrupt_init();
	timer_init();
	joypad_init();
	ppu_init();
	apu_init();
}

static const char *parse_profile_flag(int argc, char **argv) {
	for (int i = 1; i < argc; i++) {
		if ((strcmp(argv[i], "--profile") == 0 ||
		     strcmp(argv[i], "-profile") == 0) &&
		    i + 1 < argc)
			return argv[i + 1];
		if (strncmp(argv[i], "--profile=", 10) == 0)
			return argv[i] + 10;
		if (strncmp(argv[i], "-profile=", 9) == 0)
			return argv[i] + 9;
	}
	return NULL;
}

int main(int argc, char **argv) {
	const char *profilePath = parse_profile_flag(argc, argv);

	print_rom_info();
	printf("Controls: WASD/arrows = D-pad, Z = A, X = B, Enter = Start, "
	       "Backspace/Shift = Select, Esc = quit\n");

	char title[32];
	rom_title(title, sizeof(title));
	if (title[0] == '\0')
		snprintf(title, sizeof(title), "gbrc - Game Boy");

	if (!frontend_init(title)) {
		fprintf(stderr, "failed to initialize frontend\n");
		return 1;
	}

	ppu_present = frontend_present;
	apu_output = frontend_present_audio;
	gb_init();

	g_budget = CYCLES_PER_FRAME;
	unsigned long frame = 0;
	for (;;) {
		frontend_poll();
		if (frontend_should_quit())
			break;

		rom_main();
		ppu_tick();
		timer_tick();
		apu_tick();
		g_budget = cycles + CYCLES_PER_FRAME;
		frontend_wait_frame();

		frame++;
		if (profilePath && frame % 60 == 0)
			profile_dump(profilePath);
	}

	profile_dump(profilePath);

	frontend_close();
	return 0;
}
