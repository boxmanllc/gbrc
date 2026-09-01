#include "gb.h"
#include <stdio.h>

void gb_init() {
  // todooo
}

int main() {
  gb_init();
  rom_main();
  printf("cycles=%d ram[0xC000]=%02X\n", cycles, ram[0xC000]);
  return 0;
}
