#ifndef GB_APU_H
#define GB_APU_H
#include <stdint.h>

#define APU_SAMPLE_RATE 44100

void apu_init(void);
void apu_tick(void);
uint8_t apu_read(uint16_t addr);
void apu_write(uint16_t addr, uint8_t val);

extern void (*apu_output)(const int16_t *samples, int count);

#endif
