#include "gb.h"
#include "interrupt.h"
#include "joypad.h"
#include "ram.h"
#include "timer.h"
#include <stdio.h>

void gb_init() {
	interrupt_init();
	timer_init();
	joypad_init();
}

int main() {
	gb_init();
	rom_main();
	printf("cycles=%d ram[0xC000]=%02X joypad[0xFF00]=%02X\n", cycles,
	       read_ram(0xC000), read_ram(0xFF00));
	return 0;
}
