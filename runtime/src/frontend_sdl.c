#include "frontend.h"
#include "joypad.h"
#include <SDL.h>
#include <stdbool.h>
#include <stdint.h>

#define SCALE 3
#define WIN_W (160 * SCALE)
#define WIN_H (144 * SCALE)

static const uint8_t PALETTE[4][3] = {
    {0xE0, 0xF8, 0xD0},
    {0x88, 0xC0, 0x70},
    {0x34, 0x68, 0x56},
    {0x08, 0x18, 0x20},
};

static SDL_Window *g_window;
static SDL_Renderer *g_renderer;
static SDL_Texture *g_tex;
static bool g_quit_req;

static void set_button(Button b, bool down) { joypad_press(b, down); }

static void release_all_buttons(void) {
	for (int b = 0; b < 8; b++)
		joypad_press((Button)b, false);
}

static void key_event(bool down, SDL_Scancode sc) {
	switch (sc) {
	case SDL_SCANCODE_UP:
	case SDL_SCANCODE_W:
		set_button(UP, down);
		break;
	case SDL_SCANCODE_DOWN:
	case SDL_SCANCODE_S:
		set_button(DOWN, down);
		break;
	case SDL_SCANCODE_LEFT:
	case SDL_SCANCODE_A:
		set_button(LEFT, down);
		break;
	case SDL_SCANCODE_RIGHT:
	case SDL_SCANCODE_D:
		set_button(RIGHT, down);
		break;
	case SDL_SCANCODE_Z:
		set_button(A, down);
		break;
	case SDL_SCANCODE_X:
		set_button(B, down);
		break;
	case SDL_SCANCODE_RETURN:
	case SDL_SCANCODE_KP_ENTER:
		set_button(START, down);
		break;
	case SDL_SCANCODE_BACKSPACE:
	case SDL_SCANCODE_LSHIFT:
	case SDL_SCANCODE_RSHIFT:
		set_button(SELECT, down);
		break;
	case SDL_SCANCODE_ESCAPE:
		if (down)
			g_quit_req = true;
		break;
	default:
		break;
	}
}

static void pump_events(void) {
	SDL_Event ev;
	while (SDL_PollEvent(&ev)) {
		switch (ev.type) {
		case SDL_QUIT:
			g_quit_req = true;
			break;
		case SDL_KEYDOWN:
			if (ev.key.repeat == 0)
				key_event(true, ev.key.keysym.scancode);
			break;
		case SDL_KEYUP:
			key_event(false, ev.key.keysym.scancode);
			break;
		case SDL_WINDOWEVENT:
			if (ev.window.event == SDL_WINDOWEVENT_FOCUS_LOST)
				release_all_buttons();
			break;
		default:
			break;
		}
	}
}

void frontend_poll(void) { pump_events(); }

void frontend_present(const uint8_t *framebuffer) {
	if (!g_tex)
		return;

	void *pixels;
	int pitch;
	if (SDL_LockTexture(g_tex, NULL, &pixels, &pitch) != 0)
		return;

	uint8_t *dst = (uint8_t *)pixels;
	for (int y = 0; y < 144; y++) {
		for (int x = 0; x < 160; x++) {
			const uint8_t *c = PALETTE[framebuffer[y * 160 + x] & 3];
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

bool frontend_should_quit(void) { return g_quit_req; }

bool frontend_init(const char *title) {
	if (SDL_Init(SDL_INIT_VIDEO) != 0)
		return false;

	g_window =
	    SDL_CreateWindow(title, SDL_WINDOWPOS_CENTERED, SDL_WINDOWPOS_CENTERED,
	                     WIN_W, WIN_H, SDL_WINDOW_SHOWN);
	if (!g_window) {
		SDL_Quit();
		return false;
	}

	g_renderer = SDL_CreateRenderer(g_window, -1, SDL_RENDERER_ACCELERATED);
	if (!g_renderer)
		g_renderer = SDL_CreateRenderer(g_window, -1, 0);
	if (!g_renderer) {
		SDL_DestroyWindow(g_window);
		g_window = NULL;
		SDL_Quit();
		return false;
	}

	g_tex = SDL_CreateTexture(g_renderer, SDL_PIXELFORMAT_ABGR8888,
	                          SDL_TEXTUREACCESS_STREAMING, 160, 144);
	if (!g_tex) {
		SDL_DestroyRenderer(g_renderer);
		SDL_DestroyWindow(g_window);
		g_renderer = NULL;
		g_window = NULL;
		SDL_Quit();
		return false;
	}

	SDL_RaiseWindow(g_window);

	SDL_SetRenderDrawColor(g_renderer, 0, 0, 0, 255);
	SDL_RenderClear(g_renderer);
	SDL_RenderPresent(g_renderer);

	return true;
}

void frontend_close(void) {
	if (g_tex)
		SDL_DestroyTexture(g_tex);
	if (g_renderer)
		SDL_DestroyRenderer(g_renderer);
	if (g_window)
		SDL_DestroyWindow(g_window);
	SDL_Quit();
	g_tex = NULL;
	g_renderer = NULL;
	g_window = NULL;
}

void frontend_wait_frame(void) {
	static uint32_t next = 0;
	uint32_t now = SDL_GetTicks();
	if (next == 0)
		next = now;
	if (now < next) {
		SDL_Delay(next - now);
		now = next;
	}
	next = now + 16;
}
