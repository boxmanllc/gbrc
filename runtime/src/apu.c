#include "apu.h"
#include "gb.h"
#include <stdbool.h>
#include <stdint.h>
#include <string.h>

#define CPU_HZ 4194304
#define CYCLES_PER_SAMPLE (CPU_HZ / APU_SAMPLE_RATE)
#define FRAME_SEQ_PERIOD 8192
#define FLUSH_SAMPLES 1024

static const uint8_t DUTY[4][8] = {
    {0, 0, 0, 0, 0, 0, 0, 1},
    {1, 0, 0, 0, 0, 0, 0, 1},
    {1, 0, 0, 0, 0, 0, 1, 1},
    {0, 1, 1, 1, 1, 1, 1, 0},
};

static const uint8_t NOISE_DIV[8] = {8, 16, 32, 48, 64, 80, 96, 112};

typedef struct {
	bool enabled;
	bool dac;
	uint8_t duty;
	uint8_t duty_pos;
	uint16_t freq;
	int32_t timer;
	uint8_t volume;
	uint8_t env_initial;
	bool env_dir;
	uint8_t env_period;
	uint8_t env_timer;
	uint16_t length;
	bool length_enable;
} Square;

typedef struct {
	bool enabled;
	bool dac;
	uint16_t freq;
	int32_t timer;
	uint8_t pos;
	uint8_t volume_code;
	uint16_t length;
	bool length_enable;
} Wave;

typedef struct {
	bool enabled;
	bool dac;
	uint16_t lfsr;
	bool width7;
	uint8_t divisor;
	uint8_t shift;
	int32_t timer;
	uint8_t volume;
	uint8_t env_initial;
	bool env_dir;
	uint8_t env_period;
	uint8_t env_timer;
	uint16_t length;
	bool length_enable;
} Noise;

typedef struct {
	bool enabled;
	uint8_t nr50;
	uint8_t nr51;

	Square ch1;
	Square ch2;
	Wave ch3;
	Noise ch4;

	uint8_t regs[0x30];
	uint8_t wave_ram[16];

	uint8_t sweep_period;
	bool sweep_dir;
	uint8_t sweep_shift;
	uint8_t sweep_timer;
	bool sweep_enabled;

	uint32_t last_cycles;
	uint32_t sample_acc;
	uint32_t fs_acc;
	uint8_t fs_step;
} Apu;

static Apu apu;

void (*apu_output)(const int16_t *samples, int count) = 0;

void apu_init(void) {
	memset(&apu, 0, sizeof(apu));
	apu.last_cycles = cycles;
}

static uint32_t noise_period(void) {
	return (uint32_t)NOISE_DIV[apu.ch4.divisor] << apu.ch4.shift;
}

static int ch1_out(void) {
	if (!apu.ch1.enabled || !apu.ch1.dac)
		return 0;
	return DUTY[apu.ch1.duty][apu.ch1.duty_pos] ? apu.ch1.volume : 0;
}

static int ch2_out(void) {
	if (!apu.ch2.enabled || !apu.ch2.dac)
		return 0;
	return DUTY[apu.ch2.duty][apu.ch2.duty_pos] ? apu.ch2.volume : 0;
}

static int ch3_out(void) {
	if (!apu.ch3.enabled || !apu.ch3.dac || apu.ch3.volume_code == 0)
		return 0;
	uint8_t byte = apu.wave_ram[apu.ch3.pos >> 1];
	uint8_t sample = (apu.ch3.pos & 1) ? (byte & 0x0F) : (byte >> 4);
	return sample >> (apu.ch3.volume_code - 1);
}

static int ch4_out(void) {
	if (!apu.ch4.enabled || !apu.ch4.dac)
		return 0;
	return (apu.ch4.lfsr & 1) ? 0 : apu.ch4.volume;
}

static void noise_step(void) {
	uint16_t bit = (apu.ch4.lfsr ^ (apu.ch4.lfsr >> 1)) & 1;
	apu.ch4.lfsr = (uint16_t)((apu.ch4.lfsr >> 1) | (bit << 14));
	if (apu.ch4.width7)
		apu.ch4.lfsr = (uint16_t)((apu.ch4.lfsr & ~(1u << 6)) | (bit << 6));
}

static void advance(uint32_t t) {
	if (apu.ch1.enabled) {
		apu.ch1.timer -= (int32_t)t;
		while (apu.ch1.timer <= 0) {
			apu.ch1.timer += 32 * (2048 - apu.ch1.freq);
			apu.ch1.duty_pos = (uint8_t)((apu.ch1.duty_pos + 1) & 7);
		}
	}
	if (apu.ch2.enabled) {
		apu.ch2.timer -= (int32_t)t;
		while (apu.ch2.timer <= 0) {
			apu.ch2.timer += 32 * (2048 - apu.ch2.freq);
			apu.ch2.duty_pos = (uint8_t)((apu.ch2.duty_pos + 1) & 7);
		}
	}
	if (apu.ch3.enabled) {
		apu.ch3.timer -= (int32_t)t;
		while (apu.ch3.timer <= 0) {
			apu.ch3.timer += 2 * (2048 - apu.ch3.freq);
			apu.ch3.pos = (uint8_t)((apu.ch3.pos + 1) & 31);
		}
	}
	if (apu.ch4.enabled) {
		uint32_t p = noise_period();
		apu.ch4.timer -= (int32_t)t;
		while (apu.ch4.timer <= 0) {
			apu.ch4.timer += (int32_t)p;
			noise_step();
		}
	}
}

static void mix(int16_t *left, int16_t *right) {
	if (!apu.enabled) {
		*left = 0;
		*right = 0;
		return;
	}

	int v[4] = {ch1_out(), ch2_out(), ch3_out(), ch4_out()};
	int sl = 0, sr = 0;
	for (int i = 0; i < 4; i++) {
		if (apu.nr51 & (1 << i))
			sr += v[i];
		if (apu.nr51 & (1 << (i + 4)))
			sl += v[i];
	}

	int vl = (apu.nr50 >> 4) & 7;
	int vr = apu.nr50 & 7;
	*left = (int16_t)(sl * (vl + 1) * 60);
	*right = (int16_t)(sr * (vr + 1) * 60);
}

static void clock_length(void) {
	if (apu.ch1.length_enable && apu.ch1.length > 0) {
		apu.ch1.length--;
		if (apu.ch1.length == 0)
			apu.ch1.enabled = false;
	}
	if (apu.ch2.length_enable && apu.ch2.length > 0) {
		apu.ch2.length--;
		if (apu.ch2.length == 0)
			apu.ch2.enabled = false;
	}
	if (apu.ch3.length_enable && apu.ch3.length > 0) {
		apu.ch3.length--;
		if (apu.ch3.length == 0)
			apu.ch3.enabled = false;
	}
	if (apu.ch4.length_enable && apu.ch4.length > 0) {
		apu.ch4.length--;
		if (apu.ch4.length == 0)
			apu.ch4.enabled = false;
	}
}

static void clock_envelope(Square *sq) {
	if (sq->env_period == 0)
		return;
	if (sq->env_timer > 0)
		sq->env_timer--;
	if (sq->env_timer != 0)
		return;

	sq->env_timer = sq->env_period;
	if (sq->env_dir) {
		if (sq->volume < 15)
			sq->volume++;
	} else {
		if (sq->volume > 0)
			sq->volume--;
	}
}

static void clock_envelope_noise(void) {
	if (apu.ch4.env_period == 0)
		return;
	if (apu.ch4.env_timer > 0)
		apu.ch4.env_timer--;
	if (apu.ch4.env_timer != 0)
		return;

	apu.ch4.env_timer = apu.ch4.env_period;
	if (apu.ch4.env_dir) {
		if (apu.ch4.volume < 15)
			apu.ch4.volume++;
	} else {
		if (apu.ch4.volume > 0)
			apu.ch4.volume--;
	}
}

static void sweep_calc(void) {
	uint16_t delta = apu.ch1.freq >> apu.sweep_shift;
	uint16_t nf;
	if (apu.sweep_dir)
		nf = apu.ch1.freq - delta;
	else
		nf = apu.ch1.freq + delta;

	if (nf > 2047) {
		apu.ch1.enabled = false;
	} else if (apu.sweep_shift != 0) {
		apu.ch1.freq = nf;
	}
}

static void clock_sweep(void) {
	if (apu.sweep_timer > 0)
		apu.sweep_timer--;
	if (apu.sweep_timer != 0)
		return;

	apu.sweep_timer = apu.sweep_period ? apu.sweep_period : 8;
	if (!apu.sweep_enabled || apu.sweep_period == 0)
		return;

	if (apu.sweep_shift != 0)
		sweep_calc();
	sweep_calc();
}

static void frame_step(void) {
	switch (apu.fs_step) {
	case 0:
	case 2:
	case 4:
	case 6:
		clock_length();
		if (apu.fs_step == 2 || apu.fs_step == 6)
			clock_sweep();
		break;
	case 7:
		clock_envelope(&apu.ch1);
		clock_envelope(&apu.ch2);
		clock_envelope_noise();
		break;
	default:
		break;
	}

	apu.fs_step = (uint8_t)((apu.fs_step + 1) & 7);
}

void apu_tick(void) {
	uint32_t now = cycles;
	uint32_t dt_m = now - apu.last_cycles;
	apu.last_cycles = now;

	if (!apu.enabled) {
		apu.sample_acc = 0;
		apu.fs_acc = 0;
		return;
	}

	uint32_t elapsed = dt_m * 4;

	int16_t buf[FLUSH_SAMPLES];
	int n = 0;

	while (elapsed > 0) {
		uint32_t to_sample = CYCLES_PER_SAMPLE - apu.sample_acc;
		uint32_t to_fs = FRAME_SEQ_PERIOD - apu.fs_acc;
		uint32_t step = elapsed;
		if (to_sample < step)
			step = to_sample;
		if (to_fs < step)
			step = to_fs;

		apu.sample_acc += step;
		apu.fs_acc += step;
		elapsed -= step;

		advance(step);

		if (apu.sample_acc >= CYCLES_PER_SAMPLE) {
			apu.sample_acc = 0;

			int16_t l, r;
			mix(&l, &r);
			buf[n++] = l;
			buf[n++] = r;

			if (n >= FLUSH_SAMPLES) {
				if (apu_output)
					apu_output(buf, n);
				n = 0;
			}
		}

		if (apu.fs_acc >= FRAME_SEQ_PERIOD) {
			apu.fs_acc = 0;
			frame_step();
		}
	}

	if (n != 0 && apu_output)
		apu_output(buf, n);
}

uint8_t apu_read(uint16_t addr) {
	apu_tick();

	if (addr >= 0xFF30 && addr <= 0xFF3F)
		return apu.wave_ram[addr - 0xFF30];

	if (addr == 0xFF26) {
		uint8_t v = apu.enabled ? 0x80 : 0x00;
		v |= 0x70;
		if (apu.ch1.enabled)
			v |= 0x01;
		if (apu.ch2.enabled)
			v |= 0x02;
		if (apu.ch3.enabled)
			v |= 0x04;
		if (apu.ch4.enabled)
			v |= 0x08;
		return v;
	}

	uint8_t v = apu.regs[addr - 0xFF10];
	switch (addr) {
	case 0xFF10:
		return v | 0x80;
	case 0xFF11:
	case 0xFF16:
		return v | 0x3F;
	case 0xFF14:
	case 0xFF19:
	case 0xFF1E:
	case 0xFF23:
		return v | 0xBF;
	case 0xFF13:
	case 0xFF18:
	case 0xFF1B:
	case 0xFF1D:
	case 0xFF20:
		return 0xFF;
	case 0xFF1A:
		return v | 0x7F;
	case 0xFF1C:
		return v | 0x9F;
	case 0xFF1F:
	case 0xFF27:
	case 0xFF28:
	case 0xFF29:
	case 0xFF2A:
	case 0xFF2B:
	case 0xFF2C:
	case 0xFF2D:
	case 0xFF2E:
	case 0xFF2F:
		return 0xFF;
	default:
		return v;
	}
}

static void trigger_square(Square *sq, bool sweep) {
	if (sq->length == 0)
		sq->length = 64;
	sq->volume = sq->env_initial;
	sq->env_timer = sq->env_period ? sq->env_period : 8;
	sq->timer = 32 * (2048 - sq->freq);
	sq->duty_pos = 0;
	sq->enabled = sq->dac;

	if (sweep) {
		apu.sweep_timer = apu.sweep_period ? apu.sweep_period : 8;
		apu.sweep_enabled = apu.sweep_period != 0 || apu.sweep_shift != 0;
		if (apu.sweep_shift != 0)
			sweep_calc();
	}
}

static void trigger_wave(void) {
	if (apu.ch3.length == 0)
		apu.ch3.length = 256;
	apu.ch3.timer = 2 * (2048 - apu.ch3.freq);
	apu.ch3.pos = 0;
	apu.ch3.enabled = apu.ch3.dac;
}

static void trigger_noise(void) {
	if (apu.ch4.length == 0)
		apu.ch4.length = 64;
	apu.ch4.volume = apu.ch4.env_initial;
	apu.ch4.env_timer = apu.ch4.env_period ? apu.ch4.env_period : 8;
	apu.ch4.lfsr = 0x7FFF;
	apu.ch4.timer = (int32_t)noise_period();
	apu.ch4.enabled = apu.ch4.dac;
}

static void reset(void) {
	memset(&apu.ch1, 0, sizeof(apu.ch1));
	memset(&apu.ch2, 0, sizeof(apu.ch2));
	memset(&apu.ch3, 0, sizeof(apu.ch3));
	memset(&apu.ch4, 0, sizeof(apu.ch4));
	memset(apu.regs, 0, sizeof(apu.regs));
	memset(apu.wave_ram, 0, sizeof(apu.wave_ram));
	apu.nr50 = 0;
	apu.nr51 = 0;
	apu.sweep_period = 0;
	apu.sweep_dir = false;
	apu.sweep_shift = 0;
	apu.sweep_timer = 0;
	apu.sweep_enabled = false;
}

void apu_write(uint16_t addr, uint8_t val) {
	apu_tick();

	if (addr == 0xFF26) {
		bool on = (val & 0x80) != 0;
		if (on && !apu.enabled) {
			apu.enabled = true;
		} else if (!on && apu.enabled) {
			apu.enabled = false;
			reset();
		}
		return;
	}

	if (addr >= 0xFF30 && addr <= 0xFF3F) {
		apu.wave_ram[addr - 0xFF30] = val;
		return;
	}

	if (!apu.enabled)
		return;

	uint8_t i = (uint8_t)(addr - 0xFF10);
	if (i >= 0x30)
		return;
	apu.regs[i] = val;

	switch (addr) {
	case 0xFF10:
		apu.sweep_period = (val >> 4) & 7;
		apu.sweep_dir = (val & 0x08) != 0;
		apu.sweep_shift = val & 7;
		break;
	case 0xFF11:
		apu.ch1.duty = val >> 6;
		apu.ch1.length = (uint16_t)(64 - (val & 0x3F));
		break;
	case 0xFF12:
		apu.ch1.env_initial = val >> 4;
		apu.ch1.env_dir = (val & 0x08) != 0;
		apu.ch1.env_period = val & 7;
		apu.ch1.dac = (val & 0xF8) != 0;
		if (!apu.ch1.dac)
			apu.ch1.enabled = false;
		break;
	case 0xFF13:
		apu.ch1.freq = (uint16_t)((apu.ch1.freq & 0x700) | val);
		break;
	case 0xFF14:
		apu.ch1.freq = (uint16_t)((apu.ch1.freq & 0xFF) | ((val & 7) << 8));
		apu.ch1.length_enable = (val & 0x40) != 0;
		if (val & 0x80)
			trigger_square(&apu.ch1, true);
		break;
	case 0xFF16:
		apu.ch2.duty = val >> 6;
		apu.ch2.length = (uint16_t)(64 - (val & 0x3F));
		break;
	case 0xFF17:
		apu.ch2.env_initial = val >> 4;
		apu.ch2.env_dir = (val & 0x08) != 0;
		apu.ch2.env_period = val & 7;
		apu.ch2.dac = (val & 0xF8) != 0;
		if (!apu.ch2.dac)
			apu.ch2.enabled = false;
		break;
	case 0xFF18:
		apu.ch2.freq = (uint16_t)((apu.ch2.freq & 0x700) | val);
		break;
	case 0xFF19:
		apu.ch2.freq = (uint16_t)((apu.ch2.freq & 0xFF) | ((val & 7) << 8));
		apu.ch2.length_enable = (val & 0x40) != 0;
		if (val & 0x80)
			trigger_square(&apu.ch2, false);
		break;
	case 0xFF1A:
		apu.ch3.dac = (val & 0x80) != 0;
		if (!apu.ch3.dac)
			apu.ch3.enabled = false;
		break;
	case 0xFF1B:
		apu.ch3.length = (uint16_t)(256 - val);
		break;
	case 0xFF1C:
		apu.ch3.volume_code = (val >> 5) & 3;
		break;
	case 0xFF1D:
		apu.ch3.freq = (uint16_t)((apu.ch3.freq & 0x700) | val);
		break;
	case 0xFF1E:
		apu.ch3.freq = (uint16_t)((apu.ch3.freq & 0xFF) | ((val & 7) << 8));
		apu.ch3.length_enable = (val & 0x40) != 0;
		if (val & 0x80)
			trigger_wave();
		break;
	case 0xFF20:
		apu.ch4.length = (uint16_t)(64 - (val & 0x3F));
		break;
	case 0xFF21:
		apu.ch4.env_initial = val >> 4;
		apu.ch4.env_dir = (val & 0x08) != 0;
		apu.ch4.env_period = val & 7;
		apu.ch4.dac = (val & 0xF8) != 0;
		if (!apu.ch4.dac)
			apu.ch4.enabled = false;
		break;
	case 0xFF22:
		apu.ch4.shift = val >> 4;
		apu.ch4.width7 = (val & 0x08) != 0;
		apu.ch4.divisor = val & 7;
		break;
	case 0xFF23:
		apu.ch4.length_enable = (val & 0x40) != 0;
		if (val & 0x80)
			trigger_noise();
		break;
	case 0xFF24:
		apu.nr50 = val;
		break;
	case 0xFF25:
		apu.nr51 = val;
		break;
	default:
		break;
	}
}
