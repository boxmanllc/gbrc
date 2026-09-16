#include "gb.h"
#include "hardware/apu.h"
#include "hardware/joypad.h"
#include "hardware/ppu.h"
#include "hardware/timer.h"
#include "interrupt.h"

void gb_init(void) {
	interrupt_init();
	timer_init();
	joypad_init();
	ppu_init();
	apu_init();
}
