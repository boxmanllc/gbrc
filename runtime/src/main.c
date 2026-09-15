#include "frontend.h"
#include "gb.h"
#include "interrupt.h"
#include "joypad.h"
#include "ppu.h"
#include "timer.h"
#include <stdio.h>

#define CYCLES_PER_FRAME 17556

void gb_init() {
	interrupt_init();
	timer_init();
	joypad_init();
	ppu_init();
}

int main() {
	if (!frontend_init()) {
		fprintf(stderr, "failed to initialize frontend\n");
		return 1;
	}

	ppu_present = frontend_present;
	gb_init();

	g_budget = CYCLES_PER_FRAME;
	for (;;) {
		frontend_poll();
		if (frontend_should_quit())
			break;

		rom_main();
		ppu_tick();
		g_budget = cycles + CYCLES_PER_FRAME;
		frontend_wait_frame();
	}

	frontend_close();
	return 0;
}
