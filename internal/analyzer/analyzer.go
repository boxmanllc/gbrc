package analyzer

import (
	"slices"

	"github.com/0xmukesh/boxman/internal/decoder"
)

type Analyzer struct {
	decoder *decoder.Decoder
}

type Block struct {
	Start        uint16
	End          uint16
	Instructions []*decoder.Instruction
	Successors   []uint16
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
	ROM_ENTRY uint16 = 0x100
	USER_CODE uint16 = 0x150
	ROM_END   uint16 = 0x8000
)

var seeds = []uint16{
	RST_00, RST_08, RST_10, RST_18,
	RST_20, RST_28, RST_30, RST_38,
	VBLANK, LCD_STAT, TIMER, SERIAL,
	ROM_ENTRY, USER_CODE,
}

func NewAnalyzer(decoder *decoder.Decoder) *Analyzer {
	return &Analyzer{
		decoder: decoder,
	}
}

func (a *Analyzer) AnalyzeBlocks() []*Block {
	ownedBy := make([]*Block, ROM_END)
	blockStart := make([]bool, ROM_END)
	blocks := []*Block{}

	var queue []uint16
	for _, seed := range seeds {
		queue = enqueue(queue, blockStart, seed)
	}

	for len(queue) > 0 {
		start := queue[0]
		queue = queue[1:]

		if ownedBy[start] != nil {
			if ownedBy[start].Start == start {
				continue
			}

			a.cutBlock(ownedBy, ownedBy[start], start)
		}

		block := a.collectBlock(ownedBy, blockStart, start)
		if block == nil {
			continue
		}

		for _, succ := range block.Successors {
			queue = enqueue(queue, blockStart, succ)
		}

		blocks = append(blocks, block)
	}

	slices.SortFunc(blocks, func(x, y *Block) int {
		return int(x.Start) - int(y.Start)
	})

	return blocks
}

func (a *Analyzer) collectBlock(ownedBy []*Block, blockStart []bool, start uint16) *Block {
	block := &Block{Start: start}
	addr := start

	for addr < ROM_END {
		if addr != start && (ownedBy[addr] != nil || blockStart[addr]) {
			block.End = addr - 1
			block.Successors = []uint16{addr}
			return block
		}

		instr := a.decoder.DecodeAt(addr)
		if instr.InstructionType == decoder.UNKNOWN || instr.Length == 0 {
			if len(block.Instructions) == 0 {
				return nil
			}
			block.End = addr
			return block
		}

		block.Instructions = append(block.Instructions, instr)
		block.End = addr + uint16(instr.Length) - 1
		claimBytes(ownedBy, block, addr, block.End)

		if a.terminates(instr) {
			block.Successors = a.successors(addr, instr)
			return block
		}

		addr += uint16(instr.Length)
	}

	return block
}

func (a *Analyzer) cutBlock(ownedBy []*Block, block *Block, addr uint16) {
	for i := addr; i <= block.End; i++ {
		ownedBy[i] = nil
	}

	for i := 0; i < len(block.Instructions); i++ {
		if block.Instructions[i].Address >= addr {
			block.Instructions = block.Instructions[:i]
			break
		}
	}

	block.End = addr - 1
	block.Successors = []uint16{addr}
}

func claimBytes(ownedBy []*Block, block *Block, from, to uint16) {
	for i := from; i <= to; i++ {
		ownedBy[i] = block
	}
}

func enqueue(queue []uint16, blockStart []bool, addr uint16) []uint16 {
	if addr < ROM_END && !blockStart[addr] {
		blockStart[addr] = true
		return append(queue, addr)
	}
	return queue
}

func (a *Analyzer) terminates(instr *decoder.Instruction) bool {
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

func (a *Analyzer) successors(addr uint16, instr *decoder.Instruction) []uint16 {
	next := addr + uint16(instr.Length)

	successors := []uint16{}
	add := func(t uint16) {
		if t < ROM_END {
			successors = append(successors, t)
		}
	}

	switch instr.InstructionType {
	case decoder.JP_NN:
		add(instr.Imm16Bit)
	case decoder.JP_CC_NN:
		add(instr.Imm16Bit)
		add(next)
	case decoder.JR_E:
		add(a.relative(addr, instr))
	case decoder.JR_CC_E:
		add(a.relative(addr, instr))
		add(next)
	case decoder.CALL_NN, decoder.CALL_CC_NN:
		add(instr.Imm16Bit)
		add(next)
	case decoder.RET_CC:
		add(next)
	case decoder.RST_N:
		add(instr.CallFunctionAddress)
		add(next)
	}

	return successors
}

func (a *Analyzer) relative(addr uint16, instr *decoder.Instruction) uint16 {
	return uint16(int16(addr) + int16(instr.Length) + int16(int8(instr.Imm8Bit)))
}
