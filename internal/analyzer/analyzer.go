package analyzer

import (
	"github.com/0xmukesh/boxman/internal/decoder"
)

type Analyzer struct {
	decoder *decoder.Decoder
}

type Block struct {
	Start      uint16
	End        uint16
	Successors []*decoder.Instruction
}

const (
	RST_00    uint16 = 0x00
	RST_08    uint16 = 0x08
	RST_10    uint16 = 0x10
	RST_18    uint16 = 0x18
	RST_20    uint16 = 0x20
	RST_28    uint16 = 0x28
	RST_30    uint16 = 0x30
	RST_38    uint16 = 0x38
	VBLANK    uint16 = 0x40
	LCD_STAT  uint16 = 0x48
	TIMER     uint16 = 0x50
	SERIAL    uint16 = 0x58
	JOYPAD    uint16 = 0x60
	ROM_ENTRY uint16 = 0x100
	USER_CODE uint16 = 0x150
)

var seeds = []uint16{
	RST_00, RST_08, RST_10, RST_18,
	RST_20, RST_28, RST_30, RST_38,
	VBLANK, LCD_STAT, TIMER, SERIAL,
	JOYPAD, ROM_ENTRY, USER_CODE,
}

func NewAnalyzer(decoder *decoder.Decoder) *Analyzer {
	return &Analyzer{
		decoder: decoder,
	}
}

func (a *Analyzer) AnalyzeBlocks() []*Block {
	queue := seeds
	blocks := []*Block{}
	visited := make(map[uint16]bool)

	for len(queue) != 0 {
		start := queue[0]
		queue = queue[1:]

		if start >= 0x8000 || visited[start] {
			continue
		}

		block := &Block{Start: start}
		visited[start] = true
		addr := start

		for addr < 0x8000 {
			instr := a.decoder.DecodeAt(addr)
			if instr.InstructionType == decoder.UNKNOWN || instr.Length == 0 {
				block.End = addr
				break
			}

			block.End = addr + uint16(instr.Length) - 1

			if a.isBlockTerminator(instr) {
				block.Successors = a.successors(addr, instr)

				for _, succ := range block.Successors {
					if !visited[succ.Address] {
						queue = append(queue, succ.Address)
					}
				}

				break
			}

			addr += uint16(instr.Length)
		}

		blocks = append(blocks, block)
	}

	return blocks
}

func (a *Analyzer) successors(addr uint16, instr *decoder.Instruction) []*decoder.Instruction {
	next := addr + uint16(instr.Length)

	var targets []uint16
	switch instr.InstructionType {
	case decoder.JP_NN:
		targets = []uint16{instr.Imm16Bit}
	case decoder.JP_CC_NN:
		targets = []uint16{instr.Imm16Bit, next}
	case decoder.JR_E:
		targets = []uint16{a.calculateRelativeJumpTarget(addr, instr)}
	case decoder.JR_CC_E:
		targets = []uint16{a.calculateRelativeJumpTarget(addr, instr), next}
	case decoder.CALL_NN, decoder.CALL_CC_NN:
		targets = []uint16{instr.Imm16Bit, next}
	case decoder.RET_CC:
		targets = []uint16{next}
	case decoder.RST_N:
		targets = []uint16{instr.CallFunctionAddress, next}
	case decoder.JP_HL, decoder.RET, decoder.RETI:
		targets = nil
	}

	successors := []*decoder.Instruction{}
	for _, t := range targets {
		if t >= 0x8000 {
			continue
		}

		instr := a.decoder.DecodeAt(t)
		successors = append(successors, instr)
	}

	return successors
}

func (a *Analyzer) isBlockTerminator(instr *decoder.Instruction) bool {
	switch instr.InstructionType {
	case decoder.JP_NN, decoder.JP_HL, decoder.JR_E,
		decoder.RET, decoder.RETI,
		decoder.JP_CC_NN, decoder.JR_CC_E, decoder.RET_CC,
		decoder.CALL_NN, decoder.CALL_CC_NN, decoder.RST_N:
		return true
	default:
		return false
	}
}

func (a *Analyzer) calculateRelativeJumpTarget(addr uint16, instr *decoder.Instruction) uint16 {
	return uint16(int16(addr) + int16(instr.Length) + int16(int8(instr.Imm8Bit)))
}
