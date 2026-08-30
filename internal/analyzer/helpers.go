package analyzer

import "github.com/0xmukesh/boxman/internal/decoder"

func IsBlockTerminator(instr *decoder.Instruction) bool {
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
