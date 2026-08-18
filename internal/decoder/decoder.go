package decoder

import (
	"github.com/0xmukesh/boxman/internal/rom"
	"github.com/0xmukesh/boxman/internal/utils"
)

type InstructionType int
type Reg8 int

const (
	NOP InstructionType = iota
	LD_R8_R8
	LD_R8_N
	LD_R8_HL
	LD_HL_R8
	LD_HL_N
	LD_A_BC
	LD_A_DE
	LD_BC_A
	LD_DE_A
	LD_A_NN
	LD_NN_A
	LDH_A_C
	LDH_C_A
	LDH_A_N
	LDH_N_A
	LD_A_HL_DEC
	LD_HL_DEC_A
	LD_A_HL_INC
	LD_HL_INC_A
	UNKNOWN
)

const (
	Reg8B Reg8 = iota
	Reg8C
	Reg8D
	Reg8E
	Reg8H
	Reg8L
	Reg8HLIndirect
	Reg8A
)

type Instruction struct {
	InstructionType InstructionType
	Length          uint8
	MCycles         uint8
	Reg8Src         Reg8
	Reg8Dest        Reg8
	Imm8Bit         uint8
	Imm16Bit        uint16
}

type Decoder struct {
	rom *rom.Rom
}

func NewDecoder(rom *rom.Rom) *Decoder {
	return &Decoder{
		rom: rom,
	}
}

func (d *Decoder) Decode() []*Instruction {
	addr := uint16(0x0150)
	instrs := []*Instruction{}

	for addr < uint16(d.rom.RomSize) {
		opcode := d.rom.Read(addr)
		instr := d.decodeOpcode(opcode, addr)
		instrs = append(instrs, instr)
		addr += uint16(instr.Length)
	}

	return instrs
}

func (d *Decoder) decodeOpcode(opcode uint8, addr uint16) *Instruction {
	instr := &Instruction{}

	switch opcode {
	case 0x00:
		instr.InstructionType = NOP
		instr.Length = 1
		instr.MCycles = 1
	case 0x40, 0x41, 0x42, 0x43, 0x44, 0x45, 0x47,
		0x48, 0x49, 0x4A, 0x4B, 0x4C, 0x4D, 0x4F,
		0x50, 0x51, 0x52, 0x53, 0x54, 0x55, 0x57,
		0x58, 0x59, 0x5A, 0x5B, 0x5C, 0x5D, 0x5F,
		0x60, 0x61, 0x62, 0x63, 0x64, 0x65, 0x67,
		0x68, 0x69, 0x6A, 0x6B, 0x6C, 0x6D, 0x6F,
		0x78, 0x79, 0x7A, 0x7B, 0x7C, 0x7D, 0x7F:
		instr.InstructionType = LD_R8_R8
		instr.Length = 1
		instr.MCycles = 1
		instr.Reg8Src = Reg8(opcode & 0x07)
		instr.Reg8Dest = Reg8((opcode >> 3) & 0x07)
	case 0x06, 0x0E, 0x16, 0x1E,
		0x26, 0x2E, 0x3E:
		instr.InstructionType = LD_R8_N
		instr.Length = 2
		instr.MCycles = 2
		instr.Reg8Dest = Reg8((opcode >> 3) & 0x07)
		instr.Imm8Bit = d.rom.Read(addr + 1)
	case 0x46, 0x4E, 0x56, 0x5E,
		0x66, 0x6E, 0x7E:
		instr.InstructionType = LD_R8_HL
		instr.Length = 1
		instr.MCycles = 2
		instr.Reg8Dest = Reg8((opcode >> 3) & 0x07)
	case 0x70, 0x71, 0x72, 0x73,
		0x74, 0x75, 0x77:
		instr.InstructionType = LD_HL_R8
		instr.Length = 1
		instr.MCycles = 2
		instr.Reg8Src = Reg8(opcode & 0x07)
	case 0x36:
		instr.InstructionType = LD_HL_N
		instr.Length = 2
		instr.MCycles = 3
		instr.Imm8Bit = d.rom.Read(addr + 1)
	case 0x0A:
		instr.InstructionType = LD_A_BC
		instr.Length = 1
		instr.MCycles = 2
	case 0x1A:
		instr.InstructionType = LD_A_DE
		instr.Length = 1
		instr.MCycles = 2
	case 0x02:
		instr.InstructionType = LD_BC_A
		instr.Length = 1
		instr.MCycles = 2
	case 0x12:
		instr.InstructionType = LD_DE_A
		instr.Length = 1
		instr.MCycles = 2
	case 0xFA:
		instr.InstructionType = LD_A_NN
		instr.Length = 3
		instr.MCycles = 4
		instr.Imm16Bit = utils.MergeBytes(d.rom.Read(addr+1), d.rom.Read(addr+2))
	case 0xEA:
		instr.InstructionType = LD_NN_A
		instr.Length = 3
		instr.MCycles = 4
		instr.Imm16Bit = utils.MergeBytes(d.rom.Read(addr+1), d.rom.Read(addr+2))
	case 0xF2:
		instr.InstructionType = LDH_A_C
		instr.Length = 1
		instr.MCycles = 2
	case 0xE2:
		instr.InstructionType = LDH_C_A
		instr.Length = 1
		instr.MCycles = 2
	case 0xF0:
		instr.InstructionType = LDH_A_N
		instr.Length = 2
		instr.MCycles = 3
		instr.Imm8Bit = d.rom.Read(addr + 1)
	case 0xE0:
		instr.InstructionType = LDH_N_A
		instr.Length = 2
		instr.MCycles = 3
		instr.Imm8Bit = d.rom.Read(addr + 1)
	case 0x3A:
		instr.InstructionType = LD_A_HL_DEC
		instr.Length = 1
		instr.MCycles = 2
	case 0x32:
		instr.InstructionType = LD_HL_DEC_A
		instr.Length = 1
		instr.MCycles = 2
	case 0x2A:
		instr.InstructionType = LD_A_HL_INC
		instr.Length = 1
		instr.MCycles = 2
	case 0x22:
		instr.InstructionType = LD_HL_INC_A
		instr.Length = 1
		instr.MCycles = 2
	default:
		instr.InstructionType = UNKNOWN
	}

	return instr
}
