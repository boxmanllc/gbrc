#ifndef GBRC_H
#define GBRC_H

#ifdef __cplusplus
extern "C" {
#endif

#include <stdbool.h>
#include <stdint.h>

#define GB_LCD_WIDTH 160
#define GB_LCD_HEIGHT 144
#define GB_CPU_HZ 4194304
#define GB_CYCLES_PER_FRAME 17556
#define GB_SAMPLE_RATE 44100

typedef enum {
	GB_BUTTON_A = 0,
	GB_BUTTON_B,
	GB_BUTTON_SELECT,
	GB_BUTTON_START,
	GB_BUTTON_RIGHT,
	GB_BUTTON_LEFT,
	GB_BUTTON_UP,
	GB_BUTTON_DOWN,
	GB_BUTTON_COUNT,
} gb_button;

// Frontend hooks, which need to be implemented by the frontend
bool gb_init(const char *title);
void gb_shutdown(void);
void gb_poll(void);
bool gb_should_quit(void);
void gb_prepare_video(const uint8_t *framebuffer);
void gb_prepare_audio(const int16_t *samples, int count);
void gb_wait_frame(void);

// Core API functions
bool gb_attach(const char *title_override);
void gb_set_button(gb_button button, bool pressed);
void gb_set_profile_path(const char *path);
void gb_step(void);
void gb_run(void);

// Per-hardware module related tick functions
void ppu_tick(void);
void timer_tick(void);
void apu_tick(void);

// Profiling related functions
void gb_profile_tick(uint32_t frame);
void gb_profile_flush(void);

// Extern symbols which are present in the recompiled IR
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

extern uint16_t block_starts[];
extern uint32_t g_budget;
extern uint8_t IME;

uint32_t rom_main(void);
void rom_init(void);

#ifdef __cplusplus
}
#endif

#ifdef GB_IMPLEMENTATION

void gb_step(void) {
	rom_main();
	ppu_tick();
	timer_tick();
	apu_tick();
	g_budget = cycles + GB_CYCLES_PER_FRAME;
}

void gb_run(void) {
	uint32_t frame = 0;

	for (;;) {
		gb_poll();
		if (gb_should_quit()) {
			break;
		}

		gb_step();
		gb_wait_frame();

		frame++;
		gb_profile_tick(frame);
	}

	gb_profile_flush();
	gb_shutdown();
}

#endif /* GB_IMPLEMENTATION */

#endif /* GBRC_H */
