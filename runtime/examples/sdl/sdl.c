#define GB_IMPLEMENTATION
#include "gbrc.h"

#include <SDL.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdio.h>
#include <string.h>

#define SCALE 3
#define WIN_W (GB_LCD_WIDTH * SCALE)
#define WIN_H (GB_LCD_HEIGHT * SCALE)

static const uint8_t PALETTE[4][3] = {
    {0xE0, 0xF8, 0xD0},
    {0x88, 0xC0, 0x70},
    {0x34, 0x68, 0x56},
    {0x08, 0x18, 0x20},
};

static SDL_Window *g_window;
static SDL_Renderer *g_renderer;
static SDL_Texture *g_tex;
static SDL_AudioDeviceID g_audio;
static bool g_quit_req;

static void teardown(void) {
	if (g_audio) {
		SDL_CloseAudioDevice(g_audio);
		g_audio = 0;
	}
	if (g_tex) {
		SDL_DestroyTexture(g_tex);
		g_tex = NULL;
	}
	if (g_renderer) {
		SDL_DestroyRenderer(g_renderer);
		g_renderer = NULL;
	}
	if (g_window) {
		SDL_DestroyWindow(g_window);
		g_window = NULL;
	}
}

static void release_all_buttons(void) {
	for (int b = 0; b < GB_BUTTON_COUNT; b++) {
		gb_set_button((gb_button)b, false);
	}
}

static void key_event(bool down, SDL_Scancode sc) {
	switch (sc) {
	case SDL_SCANCODE_UP:
	case SDL_SCANCODE_W:
		gb_set_button(GB_BUTTON_UP, down);
		break;
	case SDL_SCANCODE_DOWN:
	case SDL_SCANCODE_S:
		gb_set_button(GB_BUTTON_DOWN, down);
		break;
	case SDL_SCANCODE_LEFT:
	case SDL_SCANCODE_A:
		gb_set_button(GB_BUTTON_LEFT, down);
		break;
	case SDL_SCANCODE_RIGHT:
	case SDL_SCANCODE_D:
		gb_set_button(GB_BUTTON_RIGHT, down);
		break;
	case SDL_SCANCODE_Z:
		gb_set_button(GB_BUTTON_A, down);
		break;
	case SDL_SCANCODE_X:
		gb_set_button(GB_BUTTON_B, down);
		break;
	case SDL_SCANCODE_RETURN:
	case SDL_SCANCODE_KP_ENTER:
		gb_set_button(GB_BUTTON_START, down);
		break;
	case SDL_SCANCODE_BACKSPACE:
	case SDL_SCANCODE_LSHIFT:
	case SDL_SCANCODE_RSHIFT:
		gb_set_button(GB_BUTTON_SELECT, down);
		break;
	case SDL_SCANCODE_ESCAPE:
		if (down) {
			g_quit_req = true;
		}
		break;
	default:
		break;
	}
}

bool gb_init(const char *title) {
	if (SDL_Init(SDL_INIT_VIDEO | SDL_INIT_AUDIO) != 0) {
		return false;
	}

	SDL_SetHint(SDL_HINT_RENDER_SCALE_QUALITY, "nearest");

	SDL_AudioSpec want, have;
	SDL_zero(want);
	want.freq = GB_SAMPLE_RATE;
	want.format = AUDIO_S16SYS;
	want.channels = 2;
	want.samples = 1024;
	want.callback = NULL;
	g_audio = SDL_OpenAudioDevice(NULL, 0, &want, &have, 0);
	if (g_audio) {
		SDL_PauseAudioDevice(g_audio, 0);
	}

	g_window =
	    SDL_CreateWindow(title, SDL_WINDOWPOS_CENTERED, SDL_WINDOWPOS_CENTERED,
	                     WIN_W, WIN_H, SDL_WINDOW_SHOWN);
	if (!g_window) {
		teardown();
		SDL_Quit();
		return false;
	}

	g_renderer = SDL_CreateRenderer(g_window, -1, SDL_RENDERER_PRESENTVSYNC);
	if (!g_renderer) {
		teardown();
		SDL_Quit();
		return false;
	}

	g_tex = SDL_CreateTexture(g_renderer, SDL_PIXELFORMAT_ABGR8888,
	                          SDL_TEXTUREACCESS_STREAMING, GB_LCD_WIDTH,
	                          GB_LCD_HEIGHT);
	if (!g_tex) {
		teardown();
		SDL_Quit();
		return false;
	}

	SDL_RaiseWindow(g_window);

	SDL_SetRenderDrawColor(g_renderer, 0, 0, 0, 255);
	SDL_RenderClear(g_renderer);
	SDL_RenderPresent(g_renderer);

	return true;
}

void gb_shutdown(void) {
	teardown();
	SDL_Quit();
}

void gb_poll(void) {
	SDL_Event ev;
	while (SDL_PollEvent(&ev)) {
		switch (ev.type) {
		case SDL_QUIT:
			g_quit_req = true;
			break;
		case SDL_KEYDOWN:
			if (ev.key.repeat == 0) {
				key_event(true, ev.key.keysym.scancode);
			}
			break;
		case SDL_KEYUP:
			key_event(false, ev.key.keysym.scancode);
			break;
		case SDL_WINDOWEVENT:
			if (ev.window.event == SDL_WINDOWEVENT_FOCUS_LOST) {
				release_all_buttons();
			}
			break;
		default:
			break;
		}
	}
}

bool gb_should_quit(void) { return g_quit_req; }

void gb_prepare_video(const uint8_t *framebuffer) {
	if (!g_tex) {
		return;
	}

	void *pixels;
	int pitch;
	if (SDL_LockTexture(g_tex, NULL, &pixels, &pitch) != 0) {
		return;
	}

	uint8_t *dst = (uint8_t *)pixels;
	for (int y = 0; y < GB_LCD_HEIGHT; y++) {
		for (int x = 0; x < GB_LCD_WIDTH; x++) {
			const uint8_t *c = PALETTE[framebuffer[y * GB_LCD_WIDTH + x] & 3];
			uint8_t *out = &dst[y * pitch + x * 4];
			out[0] = c[0];
			out[1] = c[1];
			out[2] = c[2];
			out[3] = 0xFF;
		}
	}
	SDL_UnlockTexture(g_tex);

	SDL_SetRenderDrawColor(g_renderer, 0, 0, 0, 255);
	SDL_RenderClear(g_renderer);
	SDL_RenderCopy(g_renderer, g_tex, NULL, NULL);
	SDL_RenderPresent(g_renderer);
}

void gb_prepare_audio(const int16_t *samples, int count) {
	if (!g_audio || count <= 0) {
		return;
	}

	Uint32 queued = SDL_GetQueuedAudioSize(g_audio);
	if (queued > (Uint32)(GB_SAMPLE_RATE * 2 * (int)sizeof(int16_t) / 4)) {
		return;
	}

	SDL_QueueAudio(g_audio, samples, (Uint32)(count * sizeof(int16_t)));
}

void gb_wait_frame(void) {
	static uint64_t next = 0;
	static uint64_t freq = 0;
	if (freq == 0) {
		freq = SDL_GetPerformanceFrequency();
		next = SDL_GetPerformanceCounter();
	}

	uint64_t period = freq * GB_CYCLES_PER_FRAME * 4 / GB_CPU_HZ;
	uint64_t now = SDL_GetPerformanceCounter();

	if (now > next + 2 * period) {
		next = now;
	}
	next += period;

	if (now < next) {
		uint64_t us = (next - now) * 1000000ULL / freq;
		if (us > 1500) {
			SDL_Delay((uint32_t)((us - 1500) / 1000));
		}
		while (SDL_GetPerformanceCounter() < next)
			;
	}
}

static const char *parse_profile_flag(int argc, char **argv) {
	for (int i = 1; i < argc; i++) {
		if ((strcmp(argv[i], "--profile") == 0 ||
		     strcmp(argv[i], "-profile") == 0) &&
		    i + 1 < argc) {
			return argv[i + 1];
		}

		if (strncmp(argv[i], "--profile=", 10) == 0) {
			return argv[i] + 10;
		}

		if (strncmp(argv[i], "-profile=", 9) == 0) {
			return argv[i] + 9;
		}
	}

	return NULL;
}

int main(int argc, char **argv) {
	gb_set_profile_path(parse_profile_flag(argc, argv));

	if (!gb_attach(NULL)) {
		fprintf(stderr, "failed to initialize frontend\n");
		return 1;
	}

	printf("Controls: WASD/arrows = D-pad, Z = A, X = B, Enter = Start, "
	       "Backspace/Shift = Select, Esc = quit\n");

	gb_run();
	return 0;
}
