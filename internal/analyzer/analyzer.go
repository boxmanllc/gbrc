package analyzer

import (
	"iter"
	"maps"

	"github.com/0xmukesh/boxman/internal/decoder"
)

type Analyzer struct {
	Decoder *decoder.Decoder
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

func NewAnalyzer(decoder *decoder.Decoder) *Analyzer {
	return &Analyzer{
		Decoder: decoder,
	}
}

func (a *Analyzer) FindReachableAddresses() iter.Seq[uint16] {
	queue := []uint16{
		RST_00, RST_08, RST_10, RST_18,
		RST_20, RST_28, RST_30, RST_38,
		VBLANK, LCD_STAT, TIMER, SERIAL,
		JOYPAD, ROM_ENTRY, USER_CODE,
	}
	visited := map[uint16]bool{}

	for len(queue) != 0 {
		addr := queue[0]
		queue = queue[1:]

		if visited[addr] || addr >= 0x8000 {
			continue
		}

		visited[addr] = true
		instr := a.Decoder.DecodeAt(addr)
		nextAddr := addr + uint16(instr.Length)

		switch instr.InstructionType {
		case decoder.JP_NN:
			queue = append(queue, instr.Imm16Bit)
		case decoder.JP_CC_NN:
			queue = append(queue, instr.Imm16Bit, nextAddr)
		case decoder.JR_E:
			target := int16(addr) + int16(instr.Length) + int16(int8(instr.Imm8Bit))
			queue = append(queue, uint16(target))
		case decoder.JR_CC_E:
			target := int16(addr) + int16(instr.Length) + int16(int8(instr.Imm8Bit))
			queue = append(queue, uint16(target), nextAddr)
		case decoder.CALL_NN, decoder.CALL_CC_NN:
			queue = append(queue, instr.Imm16Bit, nextAddr)
		case decoder.RST_N:
			queue = append(queue, instr.CallFunctionAddress)
		default:
			queue = append(queue, nextAddr)
		}
	}

	return maps.Keys(visited)
}
