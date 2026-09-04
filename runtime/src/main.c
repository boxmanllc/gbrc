#include "gb.h"
#include "ram.h"
#include <stdio.h>

void gb_init() {
	// todooo
}

int main() {
	gb_init();
	rom_main();
	printf("cycles=%d ram[0xC000]=%02X joypad[0xFF00]=%02X\n", cycles,
	       read_ram(0xC000), read_ram(0xFF00));
	return 0;
}
