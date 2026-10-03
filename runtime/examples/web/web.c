#define GB_IMPLEMENTATION
#include "gbrc.h"

#include <emscripten.h>
#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

// clang-format off
EM_JS(void, gbrc_js_init, (void), {
	var root = document.getElementById('emulator') || document.body;
	var canvas = document.createElement('canvas');
	canvas.width = 160;
	canvas.height = 144;
	root.appendChild(canvas);
	Module.gbrcCtx = canvas.getContext('2d');
	Module.gbrcFrame = Module.gbrcCtx.createImageData(160, 144);

	var KEYMAP = {
		"KeyX": 0, "KeyZ": 1, "ShiftLeft": 2, "ShiftRight": 2, "Enter": 3,
		"ArrowRight": 4, "ArrowLeft": 5, "ArrowUp": 6, "ArrowDown": 7,
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

	function startAudio() {
		var AC = window.AudioContext || window.webkitAudioContext;
		if (!AC) {
			return;
		}

		var SIZE = 16384;
		var a = {
			ctx: new AC({ sampleRate: 44100 }),
			l: new Float32Array(SIZE), r: new Float32Array(SIZE),
			rd: 0, wr: 0, mask: SIZE - 1, lastL: 0, lastR: 0,
		};
		Module.gbrcAudio = a;

		var node = a.ctx.createScriptProcessor(1024, 0, 2);
		node.onaudioprocess = function(e) {
			var L = e.outputBuffer.getChannelData(0);
			var R = e.outputBuffer.getChannelData(1);
			for (var i = 0; i < L.length; i++) {
				if (a.rd !== a.wr) {
					a.lastL = a.l[a.rd];
					a.lastR = a.r[a.rd];
					a.rd = (a.rd + 1) & a.mask;
				}
				L[i] = a.lastL;
				R[i] = a.lastR;
			}
		};
		node.connect(a.ctx.destination);
		a.ctx.resume();
	}

	var play = document.createElement('button');
	play.id = 'play';
	play.textContent = '▶';
	root.appendChild(play);
	play.addEventListener('click', function() {
		play.remove();
		startAudio();
		Module._gbrc_start();
	});
});

EM_JS(void, gbrc_js_audio, (const int16_t *samples, int count), {
	var a = Module.gbrcAudio;
	if (!a || a.ctx.state !== 'running') {
		return;
	}

	var MAX = 2048;
	var base = samples >> 1;
	for (var i = 0; i + 1 < count; i += 2) {
		if (((a.wr - a.rd) & a.mask) >= MAX) {
			return;
		}
		a.l[a.wr] = HEAP16[base + i] / 32768;
		a.r[a.wr] = HEAP16[base + i + 1] / 32768;
		a.wr = (a.wr + 1) & a.mask;
	}
});

EM_JS(void, gbrc_js_video, (const uint8_t *framebuffer), {
	if (!Module.gbrcPixels) {
		Module.gbrcPixels = new Uint32Array(Module.gbrcFrame.data.buffer);
		Module.gbrcPalette = new Uint32Array([
			0xFFD0F8E0, 0xFF70C088, 0xFF566834, 0xFF201808,
		]);
	}
	var px = Module.gbrcPixels, pal = Module.gbrcPalette;
	for (var i = 0; i < 160 * 144; i++) {
		px[i] = pal[HEAPU8[framebuffer + i] & 3];
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
	gbrc_js_audio(samples, count);
}

void gb_wait_frame(void) {}

static void frame(void) {
	static double next;
	const double period = 1000.0 * GB_CYCLES_PER_FRAME * 4 / GB_CPU_HZ;

	double now = emscripten_get_now();
	if (now < next - period / 4) {
		return;
	}

	next = (now - next > period) ? now + period : next + period;
	gb_step();
}

EMSCRIPTEN_KEEPALIVE
void gbrc_start(void) { emscripten_set_main_loop(frame, 0, 0); }

int main(void) { return gb_attach(NULL) ? 0 : 1; }
