#ifndef GB_H
#define GB_H
#include <stdbool.h>
#include <stdint.h>

extern uint8_t ram[0x10000];
extern uint32_t cycles;

extern uint8_t a_reg;
extern uint8_t b_reg;
extern uint8_t c_reg;
extern uint8_t d_reg;
extern uint8_t e_reg;
extern uint8_t h_reg;
extern uint8_t l_reg;

extern bool z_flag;
extern bool n_flag;
extern bool h_flag;
extern bool c_flag;

extern uint16_t pc;
extern uint16_t sp;

extern uint32_t rom_main();

void gb_init();
#endif
