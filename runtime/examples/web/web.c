// TODO: need to add audio in web example without the jittering issue

#define GB_IMPLEMENTATION
#include "gbrc.h"

#include <emscripten.h>
#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

// clang-format off

EM_JS(void, gbrc_js_init, (void), {
	var canvas = document.createElement('canvas');
	canvas.width = 160;
	canvas.height = 144;
	(document.getElementById('emulator') || document.body).appendChild(canvas);
	Module.gbrcCtx = canvas.getContext('2d');
	Module.gbrcFrame = Module.gbrcCtx.createImageData(160, 144);

	var KEYMAP = {
		'KeyX': 0, 'KeyZ': 1, 'ShiftLeft': 2, 'ShiftRight': 2, 'Enter': 3,
		'ArrowRight': 4, 'ArrowLeft': 5, 'ArrowUp': 6, 'ArrowDown': 7,
	};

	function handle(e, down) {
		var b = KEYMAP[e.code];
		if (b === undefined) {
			return;
		}

		Module._gbrc_button(b, down ? 1 : 0);
		e.preventDefault();
	}

	window.addEventListener('keydown', function(e) { handle(e, true); });
	window.addEventListener('keyup', function(e) { handle(e, false); });
});

EM_JS(void, gbrc_js_video, (const uint8_t *framebuffer), {
	var palette = [
		[0xE0, 0xF8, 0xD0], [0x88, 0xC0, 0x70],
		[0x34, 0x68, 0x56], [0x08, 0x18, 0x20],
	];
	var data = Module.gbrcFrame.data;
	for (var i = 0, j = 0; i < 160 * 144; i++, j += 4) {
		var c = palette[HEAPU8[framebuffer + i] & 3];
		data[j] = c[0];
		data[j + 1] = c[1];
		data[j + 2] = c[2];
		data[j + 3] = 0xFF;
	}
	Module.gbrcCtx.putImageData(Module.gbrcFrame, 0, 0);
});

// clang-format on

EMSCRIPTEN_KEEPALIVE
void gbrc_button(int button, int pressed) {
	gb_set_button((gb_button)button, pressed != 0);
}

bool gb_init(const char *title) {
	(void)title;

	gbrc_js_init();
	return true;
}

void gb_shutdown(void) {}

void gb_poll(void) {}

bool gb_should_quit(void) { return false; }

void gb_prepare_video(const uint8_t *framebuffer) {
	gbrc_js_video(framebuffer);
}

void gb_prepare_audio(const int16_t *samples, int count) {
	(void)samples;
	(void)count;
}

void gb_wait_frame(void) {}

int main(void) {
	if (!gb_attach(NULL)) {
		return 1;
	}

	emscripten_set_main_loop(gb_step, 0, 1);
	return 0;
}
