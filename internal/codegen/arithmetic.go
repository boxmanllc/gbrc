package codegen

import (
	"fmt"

	"github.com/0xmukesh/boxman/internal/decoder"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/enum"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

type destType int
type bit8ArithmeticOp int
type bitwiseOp int
type bitOp int

type destConfig struct {
	destType     destType
	destLocation value.Value
}

type bit8ArithmeticConfig struct {
	destConfig
	toIncludeCarryFlag bool
}

type bitwiseConfig struct {
	destConfig
	updateZeroFlag bool
}

const (
	destUnknown destType = iota
	destReg
	destHL
)

const (
	bit8OpUnknown bit8ArithmeticOp = iota
	bit8OpAdd
	bit8OpSub
	bit8OpCompare
	bit8OpIncrease
	bit8OpDecrease
	bit8OpAnd
	bit8OpOr
	bit8OpXor
)

const (
	bitwiseOpUnknown bitwiseOp = iota
	bitwiseOpRotateLeftCircular
	bitwiseOpRotateRightCircular
	bitwiseOpRotateLeft
	bitwiseOpRotateRight
	bitwiseOpShiftLeft
	bitwiseOpShiftRightArithmetic
	bitwiseOpShiftRightLogical
	bitwiseOpSwap
)

const (
	bitOpUnknown bitOp = iota
	bitOpTest
	bitOpReset
	bitOpSet
)

func (cg *Codegen) perform8BitArithmetic(
	irBlock *ir.Block, opType bit8ArithmeticOp,
	operand value.Value, cfg bit8ArithmeticConfig,
) error {
	var result, nFlag, hFlag, cFlag value.Value

	var aVal value.Value
	if opType != bit8OpIncrease && opType != bit8OpDecrease {
		a, err := cg.findReg8GlobalDef(decoder.Reg8A)
		if err != nil {
			return err
		}
		aVal = irBlock.NewLoad(types.I8, a)
	}

	var a16, operand16 value.Value
	if opType == bit8OpAdd || opType == bit8OpSub || opType == bit8OpCompare {
		a16 = irBlock.NewZExt(aVal, types.I16)
		operand16 = irBlock.NewZExt(operand, types.I16)
	}

	c16 := value.Value(constant.NewInt(types.I16, 0))
	if cfg.toIncludeCarryFlag {
		cVal := irBlock.NewLoad(types.I1, cg.cFlag)
		c16 = irBlock.NewZExt(cVal, types.I16)
	}

	switch opType {
	case bit8OpAdd:
		result16 := irBlock.NewAdd(irBlock.NewAdd(a16, operand16), c16)

		aLow := irBlock.NewAnd(a16, constant.NewInt(types.I16, 0x0F))
		operandLow := irBlock.NewAnd(operand16, constant.NewInt(types.I16, 0x0F))
		sumLow := irBlock.NewAdd(irBlock.NewAdd(aLow, operandLow), c16)

		result = irBlock.NewTrunc(result16, types.I8)
		nFlag = constant.NewInt(types.I1, 0)
		hFlag = irBlock.NewICmp(enum.IPredUGE, sumLow, constant.NewInt(types.I16, 0x10))
		cFlag = irBlock.NewICmp(enum.IPredUGE, result16, constant.NewInt(types.I16, 0x100))
	case bit8OpSub, bit8OpCompare:
		result16 := irBlock.NewSub(irBlock.NewSub(a16, operand16), c16)

		aLow := irBlock.NewAnd(a16, constant.NewInt(types.I16, 0x0F))
		operandLow := irBlock.NewAnd(operand16, constant.NewInt(types.I16, 0x0F))
		rhsLow := irBlock.NewAdd(operandLow, c16)
		rhs := irBlock.NewAdd(operand16, c16)

		result = irBlock.NewTrunc(result16, types.I8)
		nFlag = constant.NewInt(types.I1, 1)
		hFlag = irBlock.NewICmp(enum.IPredULT, aLow, rhsLow)
		cFlag = irBlock.NewICmp(enum.IPredULT, a16, rhs)
	case bit8OpIncrease:
		result = irBlock.NewAdd(operand, constant.NewInt(types.I8, 1))

		operandLow := irBlock.NewAnd(operand, constant.NewInt(types.I8, 0x0F))
		sumLow := irBlock.NewAdd(operandLow, constant.NewInt(types.I8, 1))

		nFlag = constant.NewInt(types.I1, 0)
		hFlag = irBlock.NewICmp(enum.IPredUGE, sumLow, constant.NewInt(types.I8, 0x10))
	case bit8OpDecrease:
		result = irBlock.NewSub(operand, constant.NewInt(types.I8, 1))

		operandLow := irBlock.NewAnd(operand, constant.NewInt(types.I8, 0x0F))

		nFlag = constant.NewInt(types.I1, 1)
		hFlag = irBlock.NewICmp(enum.IPredULT, operandLow, constant.NewInt(types.I8, 1))
	case bit8OpAnd:
		result = irBlock.NewAnd(aVal, operand)

		nFlag = constant.NewInt(types.I1, 0)
		hFlag = constant.NewInt(types.I1, 1)
		cFlag = constant.NewInt(types.I1, 0)
	case bit8OpOr:
		result = irBlock.NewOr(aVal, operand)

		nFlag = constant.NewInt(types.I1, 0)
		hFlag = constant.NewInt(types.I1, 0)
		cFlag = constant.NewInt(types.I1, 0)
	case bit8OpXor:
		result = irBlock.NewXor(aVal, operand)

		nFlag = constant.NewInt(types.I1, 0)
		hFlag = constant.NewInt(types.I1, 0)
		cFlag = constant.NewInt(types.I1, 0)
	}

	zFlag := irBlock.NewICmp(enum.IPredEQ, result, constant.NewInt(types.I8, 0))

	if opType != bit8OpCompare {
		switch cfg.destType {
		case destReg:
			irBlock.NewStore(result, cfg.destLocation)
		case destHL:
			cg.updateMemory(irBlock, cfg.destLocation, result)
		}
	}

	irBlock.NewStore(zFlag, cg.zFlag)
	irBlock.NewStore(nFlag, cg.nFlag)
	irBlock.NewStore(hFlag, cg.hFlag)

	if cFlag != nil {
		irBlock.NewStore(cFlag, cg.cFlag)
	}

	return nil
}

func (cg *Codegen) performBitwise(
	irBlock *ir.Block, opType bitwiseOp,
	operand value.Value, cfg bitwiseConfig,
) {
	var result, cFlag value.Value

	switch opType {
	case bitwiseOpRotateLeft:
		leftShifted := irBlock.NewShl(operand, constant.NewInt(types.I8, 1))
		lastBit := irBlock.NewLShr(operand, constant.NewInt(types.I8, 7))
		cVal := irBlock.NewLoad(types.I1, cg.cFlag)
		c8 := irBlock.NewZExt(cVal, types.I8)

		result = irBlock.NewOr(leftShifted, c8)
		cFlag = irBlock.NewTrunc(lastBit, types.I1)
	case bitwiseOpRotateLeftCircular:
		leftShifted := irBlock.NewShl(operand, constant.NewInt(types.I8, 1))
		lastBit := irBlock.NewLShr(operand, constant.NewInt(types.I8, 7))

		result = irBlock.NewOr(leftShifted, lastBit)
		cFlag = irBlock.NewTrunc(lastBit, types.I1)
	case bitwiseOpRotateRight:
		rightShifted := irBlock.NewLShr(operand, constant.NewInt(types.I8, 1))
		firstBit := irBlock.NewAnd(operand, constant.NewInt(types.I8, 1))
		cVal := irBlock.NewLoad(types.I1, cg.cFlag)
		c8 := irBlock.NewZExt(cVal, types.I8)

		result = irBlock.NewOr(
			irBlock.NewShl(c8, constant.NewInt(types.I8, 7)),
			rightShifted,
		)
		cFlag = irBlock.NewTrunc(firstBit, types.I1)
	case bitwiseOpRotateRightCircular:
		rightShifted := irBlock.NewLShr(operand, constant.NewInt(types.I8, 1))
		firstBit := irBlock.NewAnd(operand, constant.NewInt(types.I8, 1))

		result = irBlock.NewOr(
			irBlock.NewShl(firstBit, constant.NewInt(types.I8, 7)),
			rightShifted,
		)
		cFlag = irBlock.NewTrunc(firstBit, types.I1)
	case bitwiseOpShiftLeft:
		lastBit := irBlock.NewLShr(operand, constant.NewInt(types.I8, 7))

		result = irBlock.NewShl(operand, constant.NewInt(types.I8, 1))
		cFlag = irBlock.NewTrunc(lastBit, types.I1)
	case bitwiseOpShiftRightArithmetic:
		rightShifted := irBlock.NewLShr(operand, constant.NewInt(types.I8, 1))
		firstBit := irBlock.NewAnd(operand, constant.NewInt(types.I8, 1))
		lastBit := irBlock.NewAnd(operand, constant.NewInt(types.I8, 0x80))

		result = irBlock.NewOr(rightShifted, lastBit)
		cFlag = irBlock.NewTrunc(firstBit, types.I1)
	case bitwiseOpShiftRightLogical:
		firstBit := irBlock.NewAnd(operand, constant.NewInt(types.I8, 1))

		result = irBlock.NewLShr(operand, constant.NewInt(types.I8, 1))
		cFlag = irBlock.NewTrunc(firstBit, types.I1)
	case bitwiseOpSwap:
		lowNibble := irBlock.NewShl(operand, constant.NewInt(types.I8, 4))
		highNibble := irBlock.NewLShr(operand, constant.NewInt(types.I8, 4))

		result = irBlock.NewOr(lowNibble, highNibble)
		cFlag = constant.NewInt(types.I1, 0)
	}

	nFlag := constant.NewInt(types.I1, 0)
	hFlag := constant.NewInt(types.I1, 0)

	if cfg.updateZeroFlag {
		zFlag := irBlock.NewICmp(enum.IPredEQ, result, constant.NewInt(types.I8, 0))
		irBlock.NewStore(zFlag, cg.zFlag)
	}

	switch cfg.destType {
	case destReg:
		irBlock.NewStore(result, cfg.destLocation)
	case destHL:
		cg.updateMemory(irBlock, cfg.destLocation, result)
	}

	irBlock.NewStore(nFlag, cg.nFlag)
	irBlock.NewStore(hFlag, cg.hFlag)
	irBlock.NewStore(cFlag, cg.cFlag)
}

func (cg *Codegen) performBitOp(irBlock *ir.Block, opType bitOp, bitIndex, operand value.Value, cfg destConfig) {
	mask := irBlock.NewShl(constant.NewInt(types.I8, 1), bitIndex)

	switch opType {
	case bitOpTest:
		bitValue := irBlock.NewAnd(operand, mask)

		zFlag := irBlock.NewICmp(enum.IPredEQ, bitValue, constant.NewInt(types.I8, 0))
		irBlock.NewStore(zFlag, cg.zFlag)
		irBlock.NewStore(constant.NewInt(types.I1, 0), cg.nFlag)
		irBlock.NewStore(constant.NewInt(types.I1, 1), cg.hFlag)
		return
	case bitOpReset:
		notMask := irBlock.NewXor(mask, constant.NewInt(types.I8, -1))
		result := irBlock.NewAnd(operand, notMask)

		switch cfg.destType {
		case destReg:
			irBlock.NewStore(result, cfg.destLocation)
		case destHL:
			cg.updateMemory(irBlock, cfg.destLocation, result)
		}
	case bitOpSet:
		result := irBlock.NewOr(operand, mask)

		switch cfg.destType {
		case destReg:
			irBlock.NewStore(result, cfg.destLocation)
		case destHL:
			cg.updateMemory(irBlock, cfg.destLocation, result)
		}
	}
}

func (cg *Codegen) calculateOffsetFlags(irBlock *ir.Block, spVal, offsetSigned value.Value) (hFlag, cFlag value.Value) {
	result := irBlock.NewAdd(spVal, offsetSigned)
	xor := irBlock.NewXor(spVal, offsetSigned)
	xor = irBlock.NewXor(xor, result)

	hMasked := irBlock.NewAnd(xor, constant.NewInt(types.I16, 0x10))
	cMasked := irBlock.NewAnd(xor, constant.NewInt(types.I16, 0x100))

	hFlag = irBlock.NewICmp(enum.IPredNE, hMasked, constant.NewInt(types.I16, 0))
	cFlag = irBlock.NewICmp(enum.IPredNE, cMasked, constant.NewInt(types.I16, 0))
	return hFlag, cFlag
}

func (cg *Codegen) bitwiseOpFromInstrType(instr *decoder.Instruction) (bitwiseOp, error) {
	switch instr.InstructionType {
	case decoder.RLA, decoder.CB_RL_R8, decoder.CB_RL_HL:
		return bitwiseOpRotateLeft, nil
	case decoder.RLCA, decoder.CB_RLC_R8, decoder.CB_RLC_HL:
		return bitwiseOpRotateLeftCircular, nil
	case decoder.RRA, decoder.CB_RR_R8, decoder.CB_RR_HL:
		return bitwiseOpRotateRight, nil
	case decoder.RRCA, decoder.CB_RRC_R8, decoder.CB_RRC_HL:
		return bitwiseOpRotateRightCircular, nil
	case decoder.CB_SLA_R8, decoder.CB_SLA_HL:
		return bitwiseOpShiftLeft, nil
	case decoder.CB_SRA_R8, decoder.CB_SRA_HL:
		return bitwiseOpShiftRightArithmetic, nil
	case decoder.CB_SRL_R8, decoder.CB_SRL_HL:
		return bitwiseOpShiftRightLogical, nil
	case decoder.CB_SWAP_R8, decoder.CB_SWAP_HL:
		return bitwiseOpSwap, nil
	default:
		return bitwiseOpRotateLeftCircular, fmt.Errorf("%d is not a bitwise opcode", instr.InstructionType)
	}
}

func (cg *Codegen) bitOpFromInstrType(instr *decoder.Instruction) (bitOp, error) {
	switch instr.InstructionType {
	case decoder.CB_BIT_R8, decoder.CB_BIT_HL:
		return bitOpTest, nil
	case decoder.CB_RES_R8, decoder.CB_RES_HL:
		return bitOpReset, nil
	case decoder.CB_SET_R8, decoder.CB_SET_HL:
		return bitOpSet, nil
	default:
		return bitOpTest, fmt.Errorf("%d is not a bit opcode", instr.InstructionType)
	}
}

func (cg *Codegen) bit8ArithmeticInstrTypeToOpType(instr *decoder.Instruction) (bit8ArithmeticOp, error) {
	switch instr.InstructionType {
	case decoder.ADD_R8, decoder.ADD_HL, decoder.ADD_N,
		decoder.ADC_R8, decoder.ADC_HL, decoder.ADC_N:
		return bit8OpAdd, nil
	case decoder.SUB_R8, decoder.SUB_HL, decoder.SUB_N,
		decoder.SBC_R8, decoder.SBC_HL, decoder.SBC_N:
		return bit8OpSub, nil
	case decoder.CP_R8, decoder.CP_HL, decoder.CP_N:
		return bit8OpCompare, nil
	case decoder.INC_R8, decoder.INC_HL:
		return bit8OpIncrease, nil
	case decoder.DEC_R8, decoder.DEC_HL:
		return bit8OpDecrease, nil
	case decoder.AND_R8, decoder.AND_HL, decoder.AND_N:
		return bit8OpAnd, nil
	case decoder.OR_R8, decoder.OR_HL, decoder.OR_N:
		return bit8OpOr, nil
	case decoder.XOR_R8, decoder.XOR_HL, decoder.XOR_N:
		return bit8OpXor, nil
	default:
		return bit8OpAdd, fmt.Errorf("%d is not a 8-bit arithmetic opcode", instr.InstructionType)
	}
}
