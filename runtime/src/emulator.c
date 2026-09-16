#include "emulator.h"
#include "gb.h"
#include "hardware/ram.h"
#include "interrupt.h"
#include "profile.h"
#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

static uint16_t get_bc(void) { return (uint16_t)b_reg << 8 | c_reg; }
static void set_bc(uint16_t v) {
	b_reg = (uint8_t)(v >> 8);
	c_reg = (uint8_t)v;
}
static uint16_t get_de(void) { return (uint16_t)d_reg << 8 | e_reg; }
static void set_de(uint16_t v) {
	d_reg = (uint8_t)(v >> 8);
	e_reg = (uint8_t)v;
}
static uint16_t get_hl(void) { return (uint16_t)h_reg << 8 | l_reg; }
static void set_hl(uint16_t v) {
	h_reg = (uint8_t)(v >> 8);
	l_reg = (uint8_t)v;
}
static uint16_t get_af(void) {
	uint8_t f = 0;
	if (z_flag) {
		f |= 0x80;
	}
	if (n_flag) {
		f |= 0x40;
	}
	if (h_flag) {
		f |= 0x20;
	}
	if (c_flag) {
		f |= 0x10;
	}
	return (uint16_t)a_reg << 8 | f;
}
static void set_af(uint16_t v) {
	a_reg = (uint8_t)(v >> 8);
	z_flag = (v & 0x80) != 0;
	n_flag = (v & 0x40) != 0;
	h_flag = (v & 0x20) != 0;
	c_flag = (v & 0x10) != 0;
}

static uint8_t get_reg8(uint8_t enc) {
	switch (enc & 0x07) {
	case 0:
		return b_reg;
	case 1:
		return c_reg;
	case 2:
		return d_reg;
	case 3:
		return e_reg;
	case 4:
		return h_reg;
	case 5:
		return l_reg;
	case 7:
		return a_reg;
	default:
		return 0;
	}
}
static void set_reg8(uint8_t enc, uint8_t v) {
	switch (enc & 0x07) {
	case 0:
		b_reg = v;
		break;
	case 1:
		c_reg = v;
		break;
	case 2:
		d_reg = v;
		break;
	case 3:
		e_reg = v;
		break;
	case 4:
		h_reg = v;
		break;
	case 5:
		l_reg = v;
		break;
	case 7:
		a_reg = v;
		break;
	default:
		break;
	}
}

static uint16_t read16(uint16_t addr) {
	return (uint16_t)read_ram(addr) | (uint16_t)read_ram((uint16_t)(addr + 1))
	                                      << 8;
}
static void write16(uint16_t addr, uint16_t v) {
	write_ram(addr, (uint8_t)v);
	write_ram((uint16_t)(addr + 1), (uint8_t)(v >> 8));
}
static void push16(uint16_t v) {
	sp -= 2;
	write16(sp, v);
}
static uint16_t pop16(void) {
	uint16_t v = read16(sp);
	sp += 2;
	return v;
}

static bool is_block_start(uint16_t target) {
	static size_t n;
	static bool counted;
	if (!counted) {
		while (block_starts[n] != 0xFFFF) {
			n++;
		}
		counted = true;
	}
	size_t lo = 0, hi = n;
	while (lo < hi) {
		size_t mid = lo + (hi - lo) / 2;
		uint16_t v = block_starts[mid];
		if (v == target) {
			return true;
		}
		if (v < target) {
			lo = mid + 1;
		} else {
			hi = mid;
		}
	}
	return false;
}

static void alu_add(uint8_t n, bool carry) {
	uint8_t c = carry ? (c_flag ? 1 : 0) : 0;
	uint16_t r = (uint16_t)a_reg + n + c;
	h_flag = (a_reg & 0x0F) + (n & 0x0F) + c > 0x0F;
	c_flag = r > 0xFF;
	a_reg = (uint8_t)r;
	z_flag = a_reg == 0;
	n_flag = false;
}

static void alu_sub(uint8_t n, bool borrow) {
	uint8_t c = borrow ? (c_flag ? 1 : 0) : 0;
	int16_t r = (int16_t)a_reg - n - c;
	h_flag = (a_reg & 0x0F) < ((n & 0x0F) + c);
	c_flag = r < 0;
	a_reg = (uint8_t)r;
	z_flag = a_reg == 0;
	n_flag = true;
}

static void alu_and(uint8_t n) {
	a_reg &= n;
	z_flag = a_reg == 0;
	n_flag = false;
	h_flag = true;
	c_flag = false;
}
static void alu_xor(uint8_t n) {
	a_reg ^= n;
	z_flag = a_reg == 0;
	n_flag = false;
	h_flag = false;
	c_flag = false;
}
static void alu_or(uint8_t n) {
	a_reg |= n;
	z_flag = a_reg == 0;
	n_flag = false;
	h_flag = false;
	c_flag = false;
}
static void alu_cp(uint8_t n) {
	h_flag = (a_reg & 0x0F) < (n & 0x0F);
	c_flag = a_reg < n;
	z_flag = a_reg == n;
	n_flag = true;
}

static uint8_t inc8(uint8_t v) {
	h_flag = (v & 0x0F) == 0x0F;
	v++;
	z_flag = v == 0;
	n_flag = false;
	return v;
}
static uint8_t dec8(uint8_t v) {
	h_flag = (v & 0x0F) == 0;
	v--;
	z_flag = v == 0;
	n_flag = true;
	return v;
}

static void add_hl(uint16_t n) {
	uint16_t hl = get_hl();
	uint32_t r = (uint32_t)hl + n;
	h_flag = (hl & 0x0FFF) + (n & 0x0FFF) > 0x0FFF;
	c_flag = r > 0xFFFF;
	n_flag = false;
	set_hl((uint16_t)r);
}

static bool cond(uint8_t cc) {
	switch (cc) {
	case 0:
		return !z_flag;
	case 1:
		return z_flag;
	case 2:
		return !c_flag;
	case 3:
		return c_flag;
	default:
		return false;
	}
}

static void daa(void) {
	if (!n_flag) {
		if (c_flag || a_reg > 0x99) {
			a_reg += 0x60;
			c_flag = true;
		}
		if (h_flag || (a_reg & 0x0F) > 0x09) {
			a_reg += 0x06;
		}
	} else {
		if (c_flag) {
			a_reg -= 0x60;
		}
		if (h_flag) {
			a_reg -= 0x06;
		}
	}
	z_flag = a_reg == 0;
	h_flag = false;
}

static uint8_t cb_rlc(uint8_t v) {
	c_flag = (v & 0x80) != 0;
	v = (uint8_t)((v << 1) | (v >> 7));
	z_flag = v == 0;
	n_flag = false;
	h_flag = false;
	return v;
}
static uint8_t cb_rrc(uint8_t v) {
	c_flag = (v & 0x01) != 0;
	v = (uint8_t)((v >> 1) | (v << 7));
	z_flag = v == 0;
	n_flag = false;
	h_flag = false;
	return v;
}
static uint8_t cb_rl(uint8_t v) {
	uint8_t c = c_flag ? 1 : 0;
	c_flag = (v & 0x80) != 0;
	v = (uint8_t)((v << 1) | c);
	z_flag = v == 0;
	n_flag = false;
	h_flag = false;
	return v;
}
static uint8_t cb_rr(uint8_t v) {
	uint8_t c = c_flag ? 0x80 : 0;
	c_flag = (v & 0x01) != 0;
	v = (uint8_t)((v >> 1) | c);
	z_flag = v == 0;
	n_flag = false;
	h_flag = false;
	return v;
}
static uint8_t cb_sla(uint8_t v) {
	c_flag = (v & 0x80) != 0;
	v = (uint8_t)(v << 1);
	z_flag = v == 0;
	n_flag = false;
	h_flag = false;
	return v;
}
static uint8_t cb_sra(uint8_t v) {
	c_flag = (v & 0x01) != 0;
	v = (uint8_t)((v >> 1) | (v & 0x80));
	z_flag = v == 0;
	n_flag = false;
	h_flag = false;
	return v;
}
static uint8_t cb_swap(uint8_t v) {
	v = (uint8_t)((v << 4) | (v >> 4));
	z_flag = v == 0;
	n_flag = false;
	h_flag = false;
	c_flag = false;
	return v;
}
static uint8_t cb_srl(uint8_t v) {
	c_flag = (v & 0x01) != 0;
	v = (uint8_t)(v >> 1);
	z_flag = v == 0;
	n_flag = false;
	h_flag = false;
	return v;
}

static void cb_exec(uint16_t addr) {
	uint8_t cb = ram[addr];
	uint8_t reg = cb & 0x07;
	uint8_t group = cb >> 3;

	if (reg == 6) {
		cycles += 4;
	} else {
		cycles += 2;
	}

	if (group < 8) {
		uint8_t v = reg == 6 ? read_ram(get_hl()) : get_reg8(cb);
		uint8_t nv;
		switch (group) {
		case 0:
			nv = cb_rlc(v);
			break;
		case 1:
			nv = cb_rrc(v);
			break;
		case 2:
			nv = cb_rl(v);
			break;
		case 3:
			nv = cb_rr(v);
			break;
		case 4:
			nv = cb_sla(v);
			break;
		case 5:
			nv = cb_sra(v);
			break;
		case 6:
			nv = cb_swap(v);
			break;
		default:
			nv = cb_srl(v);
			break;
		}
		if (reg == 6) {
			write_ram(get_hl(), nv);
		} else {
			set_reg8(cb, nv);
		}
	} else if (group < 16) {
		uint8_t bit = group - 8;
		uint8_t v = reg == 6 ? read_ram(get_hl()) : get_reg8(cb);
		z_flag = (v & (1 << bit)) == 0;
		n_flag = false;
		h_flag = true;
	} else {
		uint8_t bit = group & 7;
		uint8_t v = reg == 6 ? read_ram(get_hl()) : get_reg8(cb);
		uint8_t nv = (group < 24) ? (uint8_t)(v & ~(1u << bit))
		                          : (uint8_t)(v | (1u << bit));
		if (reg == 6) {
			write_ram(get_hl(), nv);
		} else {
			set_reg8(cb, nv);
		}
	}
}

uint16_t interp_run(uint16_t start_pc) {
	uint16_t p = start_pc;

	profile_record(start_pc);

	for (;;) {
		if (is_block_start(p)) {
			return p;
		}
		if (cycles >= g_budget) {
			return p;
		}

		if (IME) {
			pc = p;
			uint16_t vec = interrupt_service();
			if (vec != 0xFFFF) {
				p = vec;
			}
		}

		uint8_t op = ram[p];
		p++;

		switch (op) {
		case 0x00:
			cycles += 1;
			break;
		case 0x01:
			set_bc(read16(p));
			cycles += 3;
			p += 2;
			break;
		case 0x02:
			write_ram(get_bc(), a_reg);
			cycles += 2;
			break;
		case 0x03:
			set_bc(get_bc() + 1);
			cycles += 2;
			break;
		case 0x04:
			b_reg = inc8(b_reg);
			cycles += 1;
			break;
		case 0x05:
			b_reg = dec8(b_reg);
			cycles += 1;
			break;
		case 0x06:
			b_reg = ram[p];
			cycles += 2;
			p++;
			break;
		case 0x07:
			c_flag = (a_reg & 0x80) != 0;
			a_reg = (uint8_t)((a_reg << 1) | (a_reg >> 7));
			n_flag = h_flag = false;
			cycles += 1;
			break;
		case 0x08: {
			uint16_t nn = read16(p);
			write_ram(nn, (uint8_t)sp);
			write_ram((uint16_t)(nn + 1), (uint8_t)(sp >> 8));
			cycles += 5;
			p += 2;
			break;
		}
		case 0x09:
			add_hl(get_bc());
			cycles += 2;
			break;
		case 0x0A:
			a_reg = read_ram(get_bc());
			cycles += 2;
			break;
		case 0x0B:
			set_bc(get_bc() - 1);
			cycles += 2;
			break;
		case 0x0C:
			c_reg = inc8(c_reg);
			cycles += 1;
			break;
		case 0x0D:
			c_reg = dec8(c_reg);
			cycles += 1;
			break;
		case 0x0E:
			c_reg = ram[p];
			cycles += 2;
			p++;
			break;
		case 0x0F:
			c_flag = (a_reg & 0x01) != 0;
			a_reg = (uint8_t)((a_reg >> 1) | (a_reg << 7));
			n_flag = h_flag = false;
			cycles += 1;
			break;

		case 0x10:
			cycles += 1;
			p++;
			break;
		case 0x11:
			set_de(read16(p));
			cycles += 3;
			p += 2;
			break;
		case 0x12:
			write_ram(get_de(), a_reg);
			cycles += 2;
			break;
		case 0x13:
			set_de(get_de() + 1);
			cycles += 2;
			break;
		case 0x14:
			d_reg = inc8(d_reg);
			cycles += 1;
			break;
		case 0x15:
			d_reg = dec8(d_reg);
			cycles += 1;
			break;
		case 0x16:
			d_reg = ram[p];
			cycles += 2;
			p++;
			break;
		case 0x17: {
			uint8_t c = c_flag ? 1 : 0;
			c_flag = (a_reg & 0x80) != 0;
			a_reg = (uint8_t)((a_reg << 1) | c);
			n_flag = h_flag = false;
			cycles += 1;
			break;
		}
		case 0x18:
			p = (uint16_t)((int16_t)(p + 1) + (int16_t)(int8_t)ram[p]);
			cycles += 3;
			continue;
		case 0x19:
			add_hl(get_de());
			cycles += 2;
			break;
		case 0x1A:
			a_reg = read_ram(get_de());
			cycles += 2;
			break;
		case 0x1B:
			set_de(get_de() - 1);
			cycles += 2;
			break;
		case 0x1C:
			e_reg = inc8(e_reg);
			cycles += 1;
			break;
		case 0x1D:
			e_reg = dec8(e_reg);
			cycles += 1;
			break;
		case 0x1E:
			e_reg = ram[p];
			cycles += 2;
			p++;
			break;
		case 0x1F: {
			uint8_t c = c_flag ? 0x80 : 0;
			c_flag = (a_reg & 0x01) != 0;
			a_reg = (uint8_t)((a_reg >> 1) | c);
			n_flag = h_flag = false;
			cycles += 1;
			break;
		}

		case 0x20:
		case 0x28:
		case 0x30:
		case 0x38: {
			uint8_t cc = (op >> 3) & 0x03;
			bool take = cond(cc);
			uint16_t target =
			    (uint16_t)((int16_t)(p + 1) + (int16_t)(int8_t)ram[p]);
			p++;
			cycles += 2;
			if (take) {
				cycles += 1;
				p = target;
			}
			continue;
		}
		case 0x21:
			set_hl(read16(p));
			cycles += 3;
			p += 2;
			break;
		case 0x22: {
			uint16_t hl = get_hl();
			write_ram(hl, a_reg);
			set_hl(hl + 1);
			cycles += 2;
			break;
		}
		case 0x23:
			set_hl(get_hl() + 1);
			cycles += 2;
			break;
		case 0x24:
			h_reg = inc8(h_reg);
			cycles += 1;
			break;
		case 0x25:
			h_reg = dec8(h_reg);
			cycles += 1;
			break;
		case 0x26:
			h_reg = ram[p];
			cycles += 2;
			p++;
			break;
		case 0x27:
			daa();
			cycles += 1;
			break;
		case 0x29:
			add_hl(get_hl());
			cycles += 2;
			break;
		case 0x2A: {
			uint16_t hl = get_hl();
			a_reg = read_ram(hl);
			set_hl(hl + 1);
			cycles += 2;
			break;
		}
		case 0x2B:
			set_hl(get_hl() - 1);
			cycles += 2;
			break;
		case 0x2C:
			l_reg = inc8(l_reg);
			cycles += 1;
			break;
		case 0x2D:
			l_reg = dec8(l_reg);
			cycles += 1;
			break;
		case 0x2E:
			l_reg = ram[p];
			cycles += 2;
			p++;
			break;
		case 0x2F:
			a_reg = ~a_reg;
			n_flag = h_flag = true;
			cycles += 1;
			break;

		case 0x31:
			sp = read16(p);
			cycles += 3;
			p += 2;
			break;
		case 0x32: {
			uint16_t hl = get_hl();
			write_ram(hl, a_reg);
			set_hl(hl - 1);
			cycles += 2;
			break;
		}
		case 0x33:
			sp++;
			cycles += 2;
			break;
		case 0x34: {
			uint16_t hl = get_hl();
			write_ram(hl, inc8(read_ram(hl)));
			cycles += 3;
			break;
		}
		case 0x35: {
			uint16_t hl = get_hl();
			write_ram(hl, dec8(read_ram(hl)));
			cycles += 3;
			break;
		}
		case 0x36:
			write_ram(get_hl(), ram[p]);
			cycles += 3;
			p++;
			break;
		case 0x37:
			c_flag = true;
			n_flag = h_flag = false;
			cycles += 1;
			break;
		case 0x39:
			add_hl(sp);
			cycles += 2;
			break;
		case 0x3A: {
			uint16_t hl = get_hl();
			a_reg = read_ram(hl);
			set_hl(hl - 1);
			cycles += 2;
			break;
		}
		case 0x3B:
			sp--;
			cycles += 2;
			break;
		case 0x3C:
			a_reg = inc8(a_reg);
			cycles += 1;
			break;
		case 0x3D:
			a_reg = dec8(a_reg);
			cycles += 1;
			break;
		case 0x3E:
			a_reg = ram[p];
			cycles += 2;
			p++;
			break;
		case 0x3F:
			c_flag = !c_flag;
			n_flag = h_flag = false;
			cycles += 1;
			break;

		case 0x40:
		case 0x41:
		case 0x42:
		case 0x43:
		case 0x44:
		case 0x45:
		case 0x47:
		case 0x48:
		case 0x49:
		case 0x4A:
		case 0x4B:
		case 0x4C:
		case 0x4D:
		case 0x4F:
		case 0x50:
		case 0x51:
		case 0x52:
		case 0x53:
		case 0x54:
		case 0x55:
		case 0x57:
		case 0x58:
		case 0x59:
		case 0x5A:
		case 0x5B:
		case 0x5C:
		case 0x5D:
		case 0x5F:
		case 0x60:
		case 0x61:
		case 0x62:
		case 0x63:
		case 0x64:
		case 0x65:
		case 0x67:
		case 0x68:
		case 0x69:
		case 0x6A:
		case 0x6B:
		case 0x6C:
		case 0x6D:
		case 0x6F:
		case 0x78:
		case 0x79:
		case 0x7A:
		case 0x7B:
		case 0x7C:
		case 0x7D:
		case 0x7F:
			set_reg8(op >> 3, get_reg8(op));
			cycles += 1;
			break;
		case 0x46:
		case 0x4E:
		case 0x56:
		case 0x5E:
		case 0x66:
		case 0x6E:
		case 0x7E:
			set_reg8(op >> 3, read_ram(get_hl()));
			cycles += 2;
			break;
		case 0x70:
		case 0x71:
		case 0x72:
		case 0x73:
		case 0x74:
		case 0x75:
		case 0x77:
			write_ram(get_hl(), get_reg8(op));
			cycles += 2;
			break;
		case 0x76:
			cycles += 4;
			break;

		case 0x80:
		case 0x81:
		case 0x82:
		case 0x83:
		case 0x84:
		case 0x85:
		case 0x87:
			alu_add(get_reg8(op), false);
			cycles += 1;
			break;
		case 0x86:
			alu_add(read_ram(get_hl()), false);
			cycles += 2;
			break;
		case 0x88:
		case 0x89:
		case 0x8A:
		case 0x8B:
		case 0x8C:
		case 0x8D:
		case 0x8F:
			alu_add(get_reg8(op), true);
			cycles += 1;
			break;
		case 0x8E:
			alu_add(read_ram(get_hl()), true);
			cycles += 2;
			break;
		case 0x90:
		case 0x91:
		case 0x92:
		case 0x93:
		case 0x94:
		case 0x95:
		case 0x97:
			alu_sub(get_reg8(op), false);
			cycles += 1;
			break;
		case 0x96:
			alu_sub(read_ram(get_hl()), false);
			cycles += 2;
			break;
		case 0x98:
		case 0x99:
		case 0x9A:
		case 0x9B:
		case 0x9C:
		case 0x9D:
		case 0x9F:
			alu_sub(get_reg8(op), true);
			cycles += 1;
			break;
		case 0x9E:
			alu_sub(read_ram(get_hl()), true);
			cycles += 2;
			break;
		case 0xA0:
		case 0xA1:
		case 0xA2:
		case 0xA3:
		case 0xA4:
		case 0xA5:
		case 0xA7:
			alu_and(get_reg8(op));
			cycles += 1;
			break;
		case 0xA6:
			alu_and(read_ram(get_hl()));
			cycles += 2;
			break;
		case 0xA8:
		case 0xA9:
		case 0xAA:
		case 0xAB:
		case 0xAC:
		case 0xAD:
		case 0xAF:
			alu_xor(get_reg8(op));
			cycles += 1;
			break;
		case 0xAE:
			alu_xor(read_ram(get_hl()));
			cycles += 2;
			break;
		case 0xB0:
		case 0xB1:
		case 0xB2:
		case 0xB3:
		case 0xB4:
		case 0xB5:
		case 0xB7:
			alu_or(get_reg8(op));
			cycles += 1;
			break;
		case 0xB6:
			alu_or(read_ram(get_hl()));
			cycles += 2;
			break;
		case 0xB8:
		case 0xB9:
		case 0xBA:
		case 0xBB:
		case 0xBC:
		case 0xBD:
		case 0xBF:
			alu_cp(get_reg8(op));
			cycles += 1;
			break;
		case 0xBE:
			alu_cp(read_ram(get_hl()));
			cycles += 2;
			break;

		case 0xC0:
		case 0xC8:
		case 0xD0:
		case 0xD8: {
			uint8_t cc = (op >> 3) & 0x03;
			cycles += 2;
			if (cond(cc)) {
				cycles += 3;
				p = pop16();
			}
			continue;
		}
		case 0xC1:
			set_bc(pop16());
			cycles += 3;
			break;
		case 0xC2:
		case 0xCA:
		case 0xD2:
		case 0xDA: {
			uint8_t cc = (op >> 3) & 0x03;
			uint16_t target = read16(p);
			p += 2;
			cycles += 3;
			if (cond(cc)) {
				cycles += 1;
				p = target;
			}
			continue;
		}
		case 0xC3:
			p = read16(p);
			cycles += 4;
			continue;
		case 0xC4:
		case 0xCC:
		case 0xD4:
		case 0xDC: {
			uint8_t cc = (op >> 3) & 0x03;
			uint16_t target = read16(p);
			p += 2;
			cycles += 3;
			if (cond(cc)) {
				cycles += 3;
				push16(p);
				p = target;
			}
			continue;
		}
		case 0xC5:
			push16(get_bc());
			cycles += 4;
			break;
		case 0xC6:
			alu_add(ram[p], false);
			cycles += 2;
			p++;
			break;
		case 0xC7:
		case 0xCF:
		case 0xD7:
		case 0xDF:
		case 0xE7:
		case 0xEF:
		case 0xF7:
		case 0xFF:
			push16(p);
			p = (uint16_t)(op & 0x38);
			cycles += 4;
			continue;
		case 0xC9:
			p = pop16();
			cycles += 4;
			continue;
		case 0xCB:
			cb_exec(p);
			p++;
			break;
		case 0xCD: {
			uint16_t target = read16(p);
			p += 2;
			push16(p);
			p = target;
			cycles += 6;
			continue;
		}
		case 0xCE:
			alu_add(ram[p], true);
			cycles += 2;
			p++;
			break;

		case 0xD1:
			set_de(pop16());
			cycles += 3;
			break;
		case 0xD5:
			push16(get_de());
			cycles += 4;
			break;
		case 0xD6:
			alu_sub(ram[p], false);
			cycles += 2;
			p++;
			break;
		case 0xD9:
			p = pop16();
			IME = 1;
			cycles += 4;
			continue;
		case 0xDE:
			alu_sub(ram[p], true);
			cycles += 2;
			p++;
			break;

		case 0xE0:
			write_ram((uint16_t)(0xFF00 | ram[p]), a_reg);
			cycles += 3;
			p++;
			break;
		case 0xE1:
			set_hl(pop16());
			cycles += 3;
			break;
		case 0xE2:
			write_ram((uint16_t)(0xFF00 | c_reg), a_reg);
			cycles += 2;
			break;
		case 0xE5:
			push16(get_hl());
			cycles += 4;
			break;
		case 0xE6:
			alu_and(ram[p]);
			cycles += 2;
			p++;
			break;
		case 0xE8: {
			uint8_t e8 = ram[p];
			uint16_t r = (uint16_t)((int16_t)sp + (int16_t)(int8_t)e8);
			h_flag = (sp & 0x000F) + (e8 & 0x000F) > 0x0F;
			c_flag = (sp & 0x00FF) + (e8 & 0x00FF) > 0xFF;
			sp = r;
			z_flag = n_flag = false;
			cycles += 4;
			p++;
			break;
		}
		case 0xE9:
			p = get_hl();
			cycles += 1;
			continue;
		case 0xEA:
			write_ram(read16(p), a_reg);
			cycles += 4;
			p += 2;
			break;
		case 0xEE:
			alu_xor(ram[p]);
			cycles += 2;
			p++;
			break;

		case 0xF0:
			a_reg = read_ram((uint16_t)(0xFF00 | ram[p]));
			cycles += 3;
			p++;
			break;
		case 0xF1:
			set_af(pop16());
			cycles += 3;
			break;
		case 0xF2:
			a_reg = read_ram((uint16_t)(0xFF00 | c_reg));
			cycles += 2;
			break;
		case 0xF3:
			IME = 0;
			cycles += 1;
			break;
		case 0xF5:
			push16(get_af());
			cycles += 4;
			break;
		case 0xF6:
			alu_or(ram[p]);
			cycles += 2;
			p++;
			break;
		case 0xF8: {
			uint8_t e8 = ram[p];
			uint16_t r = (uint16_t)((int16_t)sp + (int16_t)(int8_t)e8);
			h_flag = (sp & 0x000F) + (e8 & 0x000F) > 0x0F;
			c_flag = (sp & 0x00FF) + (e8 & 0x00FF) > 0xFF;
			set_hl(r);
			z_flag = n_flag = false;
			cycles += 3;
			p++;
			break;
		}
		case 0xF9:
			sp = get_hl();
			cycles += 2;
			break;
		case 0xFA:
			a_reg = read_ram(read16(p));
			cycles += 4;
			p += 2;
			break;
		case 0xFB:
			IME = 1;
			cycles += 1;
			break;
		case 0xFE:
			alu_cp(ram[p]);
			cycles += 2;
			p++;
			break;

		default:
			cycles += 1;
			break;
		}
	}
}
