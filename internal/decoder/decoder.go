package decoder

import (
	"fmt"

	"github.com/0xmukesh/boxman/internal/rom"
	"github.com/0xmukesh/boxman/internal/utils"
)

type InstructionType int
type Reg8 int
type Reg16 int
type JumpCondition int

const (
	NOP InstructionType = iota // 0
	LD_R8_R8
	LD_R8_N
	LD_R8_HL
	LD_HL_R8
	LD_HL_N
	LD_A_BC
	LD_A_DE
	LD_BC_A
	LD_DE_A
	LD_A_NN // 10

	LD_NN_A
	LDH_A_C
	LDH_C_A
	LDH_A_N
	LDH_N_A
	LD_A_HL_DEC
	LD_HL_DEC_A
	LD_A_HL_INC
	LD_HL_INC_A
	LD_R16_NN // 20

	LD_NN_SP
	LD_SP_HL
	PUSH_R16
	POP_R16
	LD_HL_SP_E
	ADD_R8
	ADD_HL
	ADD_N
	ADC_R8
	ADC_HL // 30

	ADC_N
	SUB_R8
	SUB_HL
	SUB_N
	SBC_R8
	SBC_HL
	SBC_N
	CP_R8
	CP_HL
	CP_N // 40

	INC_R8
	INC_HL
	DEC_R8
	DEC_HL
	AND_R8
	AND_HL
	AND_N
	OR_R8
	OR_HL
	OR_N // 50

	XOR_R8
	XOR_HL
	XOR_N
	CCF
	SCF
	DAA
	CPL
	INC_R16
	DEC_R16
	ADD_HL_R16 // 60

	ADD_SP_E
	RLCA
	RRCA
	RLA
	RRA
	JP_NN
	JP_HL
	JP_CC_NN
	JR_E
	JR_CC_E // 70

	CALL_NN
	CALL_CC_NN
	RET
	RET_CC
	RETI
	RST_N
	DI
	EI
	CB_RLC_R8
	CB_RLC_HL // 80

	CB_RRC_R8
	CB_RRC_HL
	CB_RL_R8
	CB_RL_HL
	CB_RR_R8
	CB_RR_HL
	CB_SLA_R8
	CB_SLA_HL
	CB_SRA_R8
	CB_SRA_HL // 90

	CB_SWAP_R8
	CB_SWAP_HL
	CB_SRL_R8
	CB_SRL_HL
	CB_BIT_R8
	CB_BIT_HL
	CB_RES_R8
	CB_RES_HL
	CB_SET_R8
	CB_SET_HL // 100

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

const (
	Reg16BC Reg16 = iota
	Reg16DE
	Reg16HL
	Reg16SP
	Reg16AF
)

const (
	NZ JumpCondition = iota
	Z
	NC
	C
)

type Instruction struct {
	Address             uint16
	Mnemonic            string
	InstructionType     InstructionType
	Length              uint8
	BaseMCycles         uint8
	AdditionalMCycles   uint8
	Reg8Src             Reg8
	Reg8Dest            Reg8
	Reg16               Reg16
	Imm8Bit             uint8
	Imm16Bit            uint16
	IsCbPrefixed        bool
	JumpCondition       JumpCondition
	CallFunctionAddress uint16
	BitOpIndex          uint8
}

type Decoder struct {
	rom   *rom.Rom
	cache map[uint16]*Instruction
}

func New(rom *rom.Rom) *Decoder {
	return &Decoder{
		rom:   rom,
		cache: make(map[uint16]*Instruction),
	}
}

func (d *Decoder) DecodeAt(addr uint16) *Instruction {
	instr, ok := d.cache[addr]
	if ok {
		return instr
	}

	opcode := d.rom.Read(addr)

	if opcode == 0xCB {
		opcode = d.rom.Read(addr + 1)
		instr = d.decodeCbPrefixedOpcode(opcode, addr)
	} else {
		instr = d.decodeOpcode(opcode, addr)
	}

	d.cache[addr] = instr
	return instr
}

func (d *Decoder) decodeOpcode(opcode uint8, addr uint16) *Instruction {
	instr := &Instruction{
		Address:      addr,
		IsCbPrefixed: false,
	}

	switch opcode {
	case 0x00:
		instr.InstructionType = NOP
		instr.Length = 1
		instr.BaseMCycles = 1
		instr.Mnemonic = "NOP"
	case 0x40, 0x41, 0x42, 0x43, 0x44, 0x45, 0x47,
		0x48, 0x49, 0x4A, 0x4B, 0x4C, 0x4D, 0x4F,
		0x50, 0x51, 0x52, 0x53, 0x54, 0x55, 0x57,
		0x58, 0x59, 0x5A, 0x5B, 0x5C, 0x5D, 0x5F,
		0x60, 0x61, 0x62, 0x63, 0x64, 0x65, 0x67,
		0x68, 0x69, 0x6A, 0x6B, 0x6C, 0x6D, 0x6F,
		0x78, 0x79, 0x7A, 0x7B, 0x7C, 0x7D, 0x7F:
		instr.InstructionType = LD_R8_R8
		instr.Length = 1
		instr.BaseMCycles = 1
		instr.Reg8Src = Reg8(opcode & 0x07)
		instr.Reg8Dest = Reg8((opcode >> 3) & 0x07)
		instr.Mnemonic = fmt.Sprintf("LD %s,%s", instr.Reg8Dest.String(), instr.Reg8Src.String())
	case 0x06, 0x0E, 0x16, 0x1E,
		0x26, 0x2E, 0x3E:
		instr.InstructionType = LD_R8_N
		instr.Length = 2
		instr.BaseMCycles = 2
		instr.Reg8Dest = Reg8((opcode >> 3) & 0x07)
		instr.Imm8Bit = d.rom.Read(addr + 1)
		instr.Mnemonic = fmt.Sprintf("LD %s,n", instr.Reg8Dest.String())
	case 0x46, 0x4E, 0x56, 0x5E,
		0x66, 0x6E, 0x7E:
		instr.InstructionType = LD_R8_HL
		instr.Length = 1
		instr.BaseMCycles = 2
		instr.Reg8Dest = Reg8((opcode >> 3) & 0x07)
		instr.Mnemonic = fmt.Sprintf("LD %s,(HL)", instr.Reg8Dest.String())
	case 0x70, 0x71, 0x72, 0x73,
		0x74, 0x75, 0x77:
		instr.InstructionType = LD_HL_R8
		instr.Length = 1
		instr.BaseMCycles = 2
		instr.Reg8Src = Reg8(opcode & 0x07)
		instr.Mnemonic = fmt.Sprintf("LD (HL),%s", instr.Reg8Src.String())
	case 0x36:
		instr.InstructionType = LD_HL_N
		instr.Length = 2
		instr.BaseMCycles = 3
		instr.Imm8Bit = d.rom.Read(addr + 1)
		instr.Mnemonic = "LD (HL),N"
	case 0x0A:
		instr.InstructionType = LD_A_BC
		instr.Length = 1
		instr.BaseMCycles = 2
		instr.Mnemonic = "LD A,(BC)"
	case 0x1A:
		instr.InstructionType = LD_A_DE
		instr.Length = 1
		instr.BaseMCycles = 2
		instr.Mnemonic = "LD A,(DE)"
	case 0x02:
		instr.InstructionType = LD_BC_A
		instr.Length = 1
		instr.BaseMCycles = 2
		instr.Mnemonic = "LD (BC),A"
	case 0x12:
		instr.InstructionType = LD_DE_A
		instr.Length = 1
		instr.BaseMCycles = 2
		instr.Mnemonic = "LD (DE),A"
	case 0xFA:
		instr.InstructionType = LD_A_NN
		instr.Length = 3
		instr.BaseMCycles = 4
		instr.Imm16Bit = utils.MergeBytes(d.rom.Read(addr+1), d.rom.Read(addr+2))
		instr.Mnemonic = "LD A,(nn)"
	case 0xEA:
		instr.InstructionType = LD_NN_A
		instr.Length = 3
		instr.BaseMCycles = 4
		instr.Imm16Bit = utils.MergeBytes(d.rom.Read(addr+1), d.rom.Read(addr+2))
		instr.Mnemonic = "LD (nn),A"
	case 0xF2:
		instr.InstructionType = LDH_A_C
		instr.Length = 1
		instr.BaseMCycles = 2
		instr.Mnemonic = "LDH A,(C)"
	case 0xE2:
		instr.InstructionType = LDH_C_A
		instr.Length = 1
		instr.BaseMCycles = 2
		instr.Mnemonic = "LDH (C),A"
	case 0xF0:
		instr.InstructionType = LDH_A_N
		instr.Length = 2
		instr.BaseMCycles = 3
		instr.Imm8Bit = d.rom.Read(addr + 1)
		instr.Mnemonic = "LDH A,(n)"
	case 0xE0:
		instr.InstructionType = LDH_N_A
		instr.Length = 2
		instr.BaseMCycles = 3
		instr.Imm8Bit = d.rom.Read(addr + 1)
		instr.Mnemonic = "LDH (n),A"
	case 0x3A:
		instr.InstructionType = LD_A_HL_DEC
		instr.Length = 1
		instr.BaseMCycles = 2
		instr.Mnemonic = "LD A,(HL-)"
	case 0x32:
		instr.InstructionType = LD_HL_DEC_A
		instr.Length = 1
		instr.BaseMCycles = 2
		instr.Mnemonic = "LD (HL-),A"
	case 0x2A:
		instr.InstructionType = LD_A_HL_INC
		instr.Length = 1
		instr.BaseMCycles = 2
		instr.Mnemonic = "LD A,(HL+)"
	case 0x22:
		instr.InstructionType = LD_HL_INC_A
		instr.Length = 1
		instr.BaseMCycles = 2
		instr.Mnemonic = "LD (HL+),A"
	case 0x01, 0x11, 0x21, 0x31:
		instr.InstructionType = LD_R16_NN
		instr.Length = 3
		instr.BaseMCycles = 3
		instr.Reg16 = Reg16((opcode >> 4) & 0x03)
		instr.Imm16Bit = utils.MergeBytes(d.rom.Read(addr+1), d.rom.Read(addr+2))
		instr.Mnemonic = fmt.Sprintf("LD %s,NN", instr.Reg16.String())
	case 0x08:
		instr.InstructionType = LD_NN_SP
		instr.Length = 3
		instr.BaseMCycles = 5
		instr.Imm16Bit = utils.MergeBytes(d.rom.Read(addr+1), d.rom.Read(addr+2))
		instr.Mnemonic = "LD NN,SP"
	case 0xF9:
		instr.InstructionType = LD_SP_HL
		instr.Length = 1
		instr.BaseMCycles = 2
		instr.Mnemonic = "LD SP,HL"
	case 0xC5, 0xD5, 0xE5, 0xF5:
		instr.InstructionType = PUSH_R16
		instr.Length = 1
		instr.BaseMCycles = 4

		reg16 := Reg16((opcode >> 4) & 0x03)
		if opcode == 0xF5 { // PUSH AF
			reg16 = Reg16AF
		}

		instr.Reg16 = reg16
		instr.Mnemonic = fmt.Sprintf("PUSH %s", instr.Reg16.String())
	case 0xC1, 0xD1, 0xE1, 0xF1:
		instr.InstructionType = POP_R16
		instr.Length = 1
		instr.BaseMCycles = 3

		reg16 := Reg16((opcode >> 4) & 0x03)
		if opcode == 0xF1 { // POP AF
			reg16 = Reg16AF
		}

		instr.Reg16 = reg16
		instr.Mnemonic = fmt.Sprintf("POP %s", instr.Reg16.String())
	case 0xF8:
		instr.InstructionType = LD_HL_SP_E
		instr.Length = 2
		instr.BaseMCycles = 3
		instr.Imm8Bit = d.rom.Read(addr + 1)
		instr.Mnemonic = "LD HL,SP+e"
	case 0x80, 0x81, 0x82, 0x83,
		0x84, 0x85, 0x87:
		instr.InstructionType = ADD_R8
		instr.Length = 1
		instr.BaseMCycles = 1
		instr.Reg8Src = Reg8(opcode & 0x07)
		instr.Mnemonic = fmt.Sprintf("ADD %s", instr.Reg8Src.String())
	case 0x86:
		instr.InstructionType = ADD_HL
		instr.Length = 1
		instr.BaseMCycles = 2
		instr.Mnemonic = "ADD (HL)"
	case 0xC6:
		instr.InstructionType = ADD_N
		instr.Length = 2
		instr.BaseMCycles = 2
		instr.Imm8Bit = d.rom.Read(addr + 1)
		instr.Mnemonic = "ADD n"
	case 0x88, 0x89, 0x8A, 0x8B,
		0x8C, 0x8D, 0x8F:
		instr.InstructionType = ADC_R8
		instr.Length = 1
		instr.BaseMCycles = 1
		instr.Reg8Src = Reg8(opcode & 0x07)
		instr.Mnemonic = fmt.Sprintf("ADC %s", instr.Reg8Src.String())
	case 0x8E:
		instr.InstructionType = ADC_HL
		instr.Length = 1
		instr.BaseMCycles = 2
		instr.Mnemonic = "ADC HL"
	case 0xCE:
		instr.InstructionType = ADC_N
		instr.Length = 2
		instr.BaseMCycles = 2
		instr.Imm8Bit = d.rom.Read(addr + 1)
		instr.Mnemonic = "ADC n"
	case 0x90, 0x91, 0x92, 0x93,
		0x94, 0x95, 0x97:
		instr.InstructionType = SUB_R8
		instr.Length = 1
		instr.BaseMCycles = 1
		instr.Reg8Src = Reg8(opcode & 0x07)
		instr.Mnemonic = fmt.Sprintf("SUB %s", instr.Reg8Src.String())
	case 0x96:
		instr.InstructionType = SUB_HL
		instr.Length = 1
		instr.BaseMCycles = 2
		instr.Mnemonic = "SUB (HL)"
	case 0xD6:
		instr.InstructionType = SUB_N
		instr.Length = 2
		instr.BaseMCycles = 2
		instr.Imm8Bit = d.rom.Read(addr + 1)
		instr.Mnemonic = "SUB n"
	case 0x98, 0x99, 0x9A, 0x9B,
		0x9C, 0x9D, 0x9F:
		instr.InstructionType = SBC_R8
		instr.Length = 1
		instr.BaseMCycles = 1
		instr.Reg8Src = Reg8(opcode & 0x07)
		instr.Mnemonic = fmt.Sprintf("SBC %s", instr.Reg8Src.String())
	case 0x9E:
		instr.InstructionType = SBC_HL
		instr.Length = 1
		instr.BaseMCycles = 2
		instr.Mnemonic = "SBC (HL)"
	case 0xDE:
		instr.InstructionType = SBC_N
		instr.Length = 2
		instr.BaseMCycles = 2
		instr.Imm8Bit = d.rom.Read(addr + 1)
		instr.Mnemonic = "SBC n"
	case 0xB8, 0xB9, 0xBA, 0xBB,
		0xBC, 0xBD, 0xBF:
		instr.InstructionType = CP_R8
		instr.Length = 1
		instr.BaseMCycles = 1
		instr.Reg8Src = Reg8(opcode & 0x07)
		instr.Mnemonic = fmt.Sprintf("CP %s", instr.Reg8Src.String())
	case 0xBE:
		instr.InstructionType = CP_HL
		instr.Length = 1
		instr.BaseMCycles = 2
		instr.Mnemonic = "CP (HL)"
	case 0xFE:
		instr.InstructionType = CP_N
		instr.Length = 2
		instr.BaseMCycles = 2
		instr.Imm8Bit = d.rom.Read(addr + 1)
		instr.Mnemonic = "CP n"
	case 0x04, 0x0C, 0x14, 0x1C,
		0x24, 0x2C, 0x3C:
		instr.InstructionType = INC_R8
		instr.Length = 1
		instr.BaseMCycles = 1
		instr.Reg8Src = Reg8((opcode >> 3) & 0x07)
		instr.Mnemonic = fmt.Sprintf("INC %s", instr.Reg8Src.String())
	case 0x34:
		instr.InstructionType = INC_HL
		instr.Length = 1
		instr.BaseMCycles = 3
		instr.Mnemonic = "INC (HL)"
	case 0x05, 0x0D, 0x15, 0x1D,
		0x25, 0x2D, 0x3D:
		instr.InstructionType = DEC_R8
		instr.Length = 1
		instr.BaseMCycles = 1
		instr.Reg8Src = Reg8((opcode >> 3) & 0x07)
		instr.Mnemonic = fmt.Sprintf("DEC %s", instr.Reg8Src.String())
	case 0x35:
		instr.InstructionType = DEC_HL
		instr.Length = 1
		instr.BaseMCycles = 3
		instr.Mnemonic = "DEC (HL)"
	case 0xA0, 0xA1, 0xA2, 0xA3,
		0xA4, 0xA5, 0xA7:
		instr.InstructionType = AND_R8
		instr.Length = 1
		instr.BaseMCycles = 1
		instr.Reg8Src = Reg8(opcode & 0x07)
		instr.Mnemonic = fmt.Sprintf("AND %s", instr.Reg8Src.String())
	case 0xA6:
		instr.InstructionType = AND_HL
		instr.Length = 1
		instr.BaseMCycles = 2
		instr.Mnemonic = "AND (HL)"
	case 0xE6:
		instr.InstructionType = AND_N
		instr.Length = 2
		instr.BaseMCycles = 2
		instr.Imm8Bit = d.rom.Read(addr + 1)
		instr.Mnemonic = "AND n"
	case 0xB0, 0xB1, 0xB2, 0xB3,
		0xB4, 0xB5, 0xB7:
		instr.InstructionType = OR_R8
		instr.Length = 1
		instr.BaseMCycles = 1
		instr.Reg8Src = Reg8(opcode & 0x07)
		instr.Mnemonic = fmt.Sprintf("OR %s", instr.Reg8Src.String())
	case 0xB6:
		instr.InstructionType = OR_HL
		instr.Length = 1
		instr.BaseMCycles = 2
		instr.Mnemonic = "OR (HL)"
	case 0xF6:
		instr.InstructionType = OR_N
		instr.Length = 2
		instr.BaseMCycles = 2
		instr.Imm8Bit = d.rom.Read(addr + 1)
		instr.Mnemonic = "OR n"
	case 0xA8, 0xA9, 0xAA, 0xAB,
		0xAC, 0xAD, 0xAF:
		instr.InstructionType = XOR_R8
		instr.Length = 1
		instr.BaseMCycles = 1
		instr.Reg8Src = Reg8(opcode & 0x07)
		instr.Mnemonic = fmt.Sprintf("XOR %s", instr.Reg8Src.String())
	case 0xAE:
		instr.InstructionType = XOR_HL
		instr.Length = 1
		instr.BaseMCycles = 2
		instr.Mnemonic = "XOR (HL)"
	case 0xEE:
		instr.InstructionType = XOR_N
		instr.Length = 2
		instr.BaseMCycles = 2
		instr.Imm8Bit = d.rom.Read(addr + 1)
		instr.Mnemonic = "XOR n"
	case 0x3F:
		instr.InstructionType = CCF
		instr.Length = 1
		instr.BaseMCycles = 1
		instr.Mnemonic = "CCF"
	case 0x37:
		instr.InstructionType = SCF
		instr.Length = 1
		instr.BaseMCycles = 1
		instr.Mnemonic = "SCF"
	case 0x27:
		instr.InstructionType = DAA
		instr.Length = 1
		instr.BaseMCycles = 1
		instr.Mnemonic = "DAA"
	case 0x2F:
		instr.InstructionType = CPL
		instr.Length = 1
		instr.BaseMCycles = 1
		instr.Mnemonic = "CPL"
	case 0x03, 0x13, 0x23, 0x33:
		instr.InstructionType = INC_R16
		instr.Length = 1
		instr.BaseMCycles = 2
		instr.Reg16 = Reg16((opcode >> 4) & 0x03)
		instr.Mnemonic = fmt.Sprintf("INC %s", instr.Reg16.String())
	case 0x0B, 0x1B, 0x2B, 0x3B:
		instr.InstructionType = DEC_R16
		instr.Length = 1
		instr.BaseMCycles = 2
		instr.Reg16 = Reg16((opcode >> 4) & 0x03)
		instr.Mnemonic = fmt.Sprintf("DEC %s", instr.Reg16.String())
	case 0x09, 0x19, 0x29, 0x39:
		instr.InstructionType = ADD_HL_R16
		instr.Length = 1
		instr.BaseMCycles = 2
		instr.Reg16 = Reg16((opcode >> 4) & 0x03)
		instr.Mnemonic = fmt.Sprintf("ADD HL,%s", instr.Reg16.String())
	case 0xE8:
		instr.InstructionType = ADD_SP_E
		instr.Length = 2
		instr.BaseMCycles = 4
		instr.Imm8Bit = d.rom.Read(addr + 1)
		instr.Mnemonic = "ADD SP,e"
	case 0x07:
		instr.InstructionType = RLCA
		instr.Length = 1
		instr.BaseMCycles = 1
		instr.Mnemonic = "RLCA"
	case 0x0F:
		instr.InstructionType = RRCA
		instr.Length = 1
		instr.BaseMCycles = 1
		instr.Mnemonic = "RRCA"
	case 0x17:
		instr.InstructionType = RLA
		instr.Length = 1
		instr.BaseMCycles = 1
		instr.Mnemonic = "RLA"
	case 0x1F:
		instr.InstructionType = RRA
		instr.Length = 1
		instr.BaseMCycles = 1
		instr.Mnemonic = "RRA"
	case 0xC3:
		instr.InstructionType = JP_NN
		instr.Length = 3
		instr.BaseMCycles = 4
		instr.Imm16Bit = utils.MergeBytes(d.rom.Read(addr+1), d.rom.Read(addr+2))
		instr.Mnemonic = "JP nn"
	case 0xE9:
		instr.InstructionType = JP_HL
		instr.Length = 1
		instr.BaseMCycles = 1
		instr.Mnemonic = "JP (HL)"
	case 0xC2, 0xCA, 0xD2, 0xDA:
		instr.InstructionType = JP_CC_NN
		instr.Length = 3
		instr.BaseMCycles = 3
		instr.AdditionalMCycles = 1
		instr.Imm16Bit = utils.MergeBytes(d.rom.Read(addr+1), d.rom.Read(addr+2))
		instr.JumpCondition = JumpCondition((opcode >> 3) & 0x03)
		instr.Mnemonic = fmt.Sprintf("JP %s,nn", instr.JumpCondition.String())
	case 0x18:
		instr.InstructionType = JR_E
		instr.Length = 2
		instr.BaseMCycles = 3
		instr.Imm8Bit = d.rom.Read(addr + 1)
		instr.Mnemonic = "JR e"
	case 0x20, 0x28, 0x30, 0x38:
		instr.InstructionType = JR_CC_E
		instr.Length = 2
		instr.BaseMCycles = 2
		instr.AdditionalMCycles = 1
		instr.Imm8Bit = d.rom.Read(addr + 1)
		instr.JumpCondition = JumpCondition((opcode >> 3) & 0x03)
		instr.Mnemonic = fmt.Sprintf("JR %s,e", instr.JumpCondition.String())
	case 0xCD:
		instr.InstructionType = CALL_NN
		instr.Length = 3
		instr.BaseMCycles = 6
		instr.Imm16Bit = utils.MergeBytes(d.rom.Read(addr+1), d.rom.Read(addr+2))
		instr.Mnemonic = "CALL nn"
	case 0xC4, 0xCC, 0xD4, 0xDC:
		instr.InstructionType = CALL_CC_NN
		instr.Length = 3
		instr.BaseMCycles = 3
		instr.AdditionalMCycles = 3
		instr.Imm16Bit = utils.MergeBytes(d.rom.Read(addr+1), d.rom.Read(addr+2))
		instr.JumpCondition = JumpCondition((opcode >> 3) & 0x03)
		instr.Mnemonic = fmt.Sprintf("CALL %s,nn", instr.JumpCondition.String())
	case 0xC9:
		instr.InstructionType = RET
		instr.Length = 1
		instr.BaseMCycles = 4
		instr.Mnemonic = "RET"
	case 0xC0, 0xC8, 0xD0, 0xD8:
		instr.InstructionType = RET_CC
		instr.Length = 1
		instr.BaseMCycles = 2
		instr.AdditionalMCycles = 3
		instr.JumpCondition = JumpCondition((opcode >> 3) & 0x03)
		instr.Mnemonic = fmt.Sprintf("RET %s", instr.JumpCondition.String())
	case 0xD9:
		instr.InstructionType = RETI
		instr.Length = 1
		instr.BaseMCycles = 4
		instr.Mnemonic = "RETI"
	case 0xC7, 0xCF, 0xD7, 0xDF,
		0xE7, 0xEF, 0xF7, 0xFF:
		instr.InstructionType = RST_N
		instr.Length = 1
		instr.BaseMCycles = 4
		instr.CallFunctionAddress = [8]uint16{
			0x00, 0x08, 0x10, 0x18,
			0x20, 0x28, 0x30, 0x38,
		}[(opcode>>3)&0x07]
		instr.Mnemonic = fmt.Sprintf("RST 0x%02X", instr.CallFunctionAddress)
	case 0xF3:
		instr.InstructionType = DI
		instr.Length = 1
		instr.BaseMCycles = 1
		instr.Mnemonic = "DI"
	case 0xFB:
		instr.InstructionType = EI
		instr.Length = 1
		instr.BaseMCycles = 1
		instr.Mnemonic = "EI"
	default:
		instr.InstructionType = UNKNOWN
	}

	return instr
}

func (d *Decoder) decodeCbPrefixedOpcode(opcode uint8, addr uint16) *Instruction {
	instr := &Instruction{
		Address:      addr,
		IsCbPrefixed: true,
	}

	var narrowBlockOps = [][2]InstructionType{
		{CB_RLC_R8, CB_RLC_HL},
		{CB_RRC_R8, CB_RRC_HL},
		{CB_RL_R8, CB_RL_HL},
		{CB_RR_R8, CB_RR_HL},
		{CB_SLA_R8, CB_SLA_HL},
		{CB_SRA_R8, CB_SRA_HL},
		{CB_SWAP_R8, CB_SWAP_HL},
		{CB_SRL_R8, CB_SRL_HL},
	}

	var wideBlockOps = [][2]InstructionType{
		{CB_BIT_R8, CB_BIT_HL},
		{CB_RES_R8, CB_RES_HL},
		{CB_SET_R8, CB_SET_HL},
	}

	narrowBlockMnemonic := []string{"RLC", "RRC", "RL", "RR", "SLA", "SRA", "SWAP", "SRL"}
	wideBlockMnemonic := []string{"BIT", "RES", "SET"}

	reg := opcode & 0x07
	var group [2]InstructionType
	var baseMnemonic string

	if opcode < 0x40 {
		idx := opcode >> 3
		group = narrowBlockOps[idx]
		baseMnemonic = narrowBlockMnemonic[idx]
	} else {
		idx := (opcode - 0x40) / 0x40
		group = wideBlockOps[idx]
		baseMnemonic = wideBlockMnemonic[idx]
		instr.BitOpIndex = (opcode >> 3) & 0x07
	}

	if reg == 0x06 {
		instr.InstructionType = group[1]
		instr.Length = 2
		instr.BaseMCycles = 4
		instr.Mnemonic = fmt.Sprintf("%s (HL)", baseMnemonic)
	} else {
		instr.InstructionType = group[0]
		instr.Length = 2
		instr.BaseMCycles = 2
		instr.Reg8Src = Reg8(reg)
		instr.Mnemonic = fmt.Sprintf("%s %s", baseMnemonic, instr.Reg8Src.String())
	}

	return instr
}

func (r Reg8) String() string {
	switch r {
	case Reg8B:
		return "B"
	case Reg8C:
		return "C"
	case Reg8D:
		return "D"
	case Reg8E:
		return "E"
	case Reg8H:
		return "H"
	case Reg8L:
		return "L"
	case Reg8HLIndirect:
		return "HL"
	case Reg8A:
		return "A"
	default:
		panic(fmt.Sprintf("unknown reg8: %d", r))
	}
}

func (r Reg16) String() string {
	switch r {
	case Reg16BC:
		return "BC"
	case Reg16DE:
		return "DE"
	case Reg16HL:
		return "HL"
	case Reg16SP:
		return "SP"
	case Reg16AF:
		return "AF"
	default:
		panic(fmt.Sprintf("unknown reg16: %d", r))
	}
}

func (c JumpCondition) String() string {
	switch c {
	case NZ:
		return "NZ"
	case Z:
		return "Z"
	case NC:
		return "NC"
	case C:
		return "C"
	default:
		panic(fmt.Sprintf("unknown jump condition: %d", c))
	}
}
