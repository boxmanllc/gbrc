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
type rotateOp int

type destConfig struct {
	destType     destType
	destLocation value.Value
}

type bit8ArithmeticConfig struct {
	destConfig
	toIncludeCarryFlag bool // whether to include carry flag within operations
}

type rotateConfig struct {
	destConfig
	updateZeroFlag bool // whether to set zero flag based on the final result
}

const (
	destReg destType = iota // store result in a register
	destHL                  // store result in location pointed by (HL)
)

const (
	bit8OpAdd bit8ArithmeticOp = iota
	bit8OpSub
	bit8OpCompare
	bit8OpIncrease
	bit8OpDecrease
	bit8OpAnd
	bit8OpOr
	bit8OpXor
)

const (
	rotateOpLeftCircular rotateOp = iota
	rotateOpRightCircular
	rotateOpLeft
	rotateOpRight
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
		// stores result in A register
		// flags:
		// 	 z: result == 0
		// 	 n: 0
		// 	 h: (a & 0x0F) + (b & 0x0F) + c >= 0x10
		// 	 c: result16 >= 0x100
		result16 := irBlock.NewAdd(ir.NewAdd(a16, operand16), c16)

		aLow := irBlock.NewAnd(a16, constant.NewInt(types.I16, 0x0F))
		operandLow := irBlock.NewAnd(operand16, constant.NewInt(types.I16, 0x0F))
		sumLow := irBlock.NewAdd(irBlock.NewAdd(aLow, operandLow), c16)

		result = irBlock.NewTrunc(result16, types.I8)
		nFlag = constant.NewInt(types.I1, 0)
		hFlag = irBlock.NewICmp(enum.IPredUGE, sumLow, constant.NewInt(types.I16, 0x10))
		cFlag = irBlock.NewICmp(enum.IPredUGE, result16, constant.NewInt(types.I16, 0x100))
	case bit8OpSub, bit8OpCompare:
		// stores result in A register
		// flags:
		//   z: result == 0
		//   n: 1
		//   h: (a & 0x0F) < (b & 0x0F) + c
		//   c: a < b + c
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
		// stores result in either source register or location pointed by (HL)
		// flags:
		//   z: result == 0
		//   n: 0
		//   h: (operand & 0x0F) + 1 >= 0x10
		result = irBlock.NewAdd(operand, constant.NewInt(types.I8, 1))

		operandLow := irBlock.NewAnd(operand, constant.NewInt(types.I8, 0x0F))
		sumLow := irBlock.NewAdd(operandLow, constant.NewInt(types.I8, 1))

		nFlag = constant.NewInt(types.I1, 0)
		hFlag = irBlock.NewICmp(enum.IPredUGE, sumLow, constant.NewInt(types.I8, 0x10))
	case bit8OpDecrease:
		// stores result in either source register or location point by (HL)
		// flags:
		//   z: result == 0
		//   n: 1
		//   h: (operand & 0x0F) < 1
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

func (cg *Codegen) performRotate(
	irBlock *ir.Block, opType rotateOp,
	operand value.Value, cfg rotateConfig,
) {
	var result value.Value
	var zFlag, nFlag, hFlag, cFlag value.Value

	switch opType {
	case rotateOpLeft:
		leftShifted := irBlock.NewShl(operand, constant.NewInt(types.I8, 1))
		lastBit := irBlock.NewLShr(operand, constant.NewInt(types.I8, 7))
		cVal := irBlock.NewLoad(types.I1, cg.cFlag)
		c8 := irBlock.NewZExt(cVal, types.I8)

		result = irBlock.NewOr(leftShifted, c8)
		cFlag = irBlock.NewTrunc(lastBit, types.I1)
	case rotateOpLeftCircular:
		leftShifted := irBlock.NewShl(operand, constant.NewInt(types.I8, 1))
		lastBit := irBlock.NewLShr(operand, constant.NewInt(types.I8, 7))

		result = irBlock.NewOr(leftShifted, lastBit)
		cFlag = irBlock.NewTrunc(lastBit, types.I1)
	case rotateOpRight:
		rightShifted := irBlock.NewLShr(operand, constant.NewInt(types.I8, 1))
		firstBit := irBlock.NewAnd(operand, constant.NewInt(types.I8, 1))
		cVal := irBlock.NewLoad(types.I1, cg.cFlag)
		c8 := irBlock.NewZExt(cVal, types.I8)

		result = irBlock.NewOr(
			irBlock.NewShl(c8, constant.NewInt(types.I8, 7)),
			rightShifted,
		)
		cFlag = irBlock.NewTrunc(firstBit, types.I1)
	case rotateOpRightCircular:
		rightShifted := irBlock.NewLShr(operand, constant.NewInt(types.I8, 1))
		firstBit := irBlock.NewAnd(operand, constant.NewInt(types.I8, 1))

		result = irBlock.NewOr(
			irBlock.NewShl(firstBit, constant.NewInt(types.I8, 7)),
			rightShifted,
		)
		cFlag = irBlock.NewTrunc(firstBit, types.I1)
	}

	if cfg.updateZeroFlag {
		zFlag = irBlock.NewICmp(enum.IPredEQ, result, constant.NewInt(types.I8, 0))
	}

	nFlag = constant.NewInt(types.I1, 0)
	hFlag = constant.NewInt(types.I1, 0)

	switch cfg.destType {
	case destReg:
		irBlock.NewStore(result, cfg.destLocation)
	case destHL:
		cg.updateMemory(irBlock, cfg.destLocation, result)
	}

	irBlock.NewStore(zFlag, cg.zFlag)
	irBlock.NewStore(nFlag, cg.nFlag)
	irBlock.NewStore(hFlag, cg.hFlag)
	irBlock.NewStore(cFlag, cg.cFlag)
}

func (cg *Codegen) calculateOffsetFlags(irBlock *ir.Block, spVal, offset value.Value) (hFlag, cFlag value.Value) {
	spLow := irBlock.NewAnd(spVal, constant.NewInt(types.I16, 0x0F))
	offsetLow := irBlock.NewAnd(offset, constant.NewInt(types.I16, 0x0F))
	lowSum := irBlock.NewAdd(spLow, offsetLow)

	spByte := irBlock.NewAnd(spVal, constant.NewInt(types.I16, 0xFF))
	byteSum := irBlock.NewAdd(spByte, offset)

	hFlag = irBlock.NewICmp(enum.IPredUGE, lowSum, constant.NewInt(types.I16, 0x10))
	cFlag = irBlock.NewICmp(enum.IPredUGE, byteSum, constant.NewInt(types.I16, 0x100))
	return hFlag, cFlag
}

func (cg *Codegen) bit8ArithmeticInstrTypeToOpType(instr *decoder.Instruction) bit8ArithmeticOp {
	switch instr.InstructionType {
	case decoder.ADD_R8, decoder.ADD_HL, decoder.ADD_N,
		decoder.ADC_R8, decoder.ADC_HL, decoder.ADC_N:
		return bit8OpAdd
	case decoder.SUB_R8, decoder.SUB_HL, decoder.SUB_N,
		decoder.SBC_R8, decoder.SBC_HL, decoder.SBC_N:
		return bit8OpSub
	case decoder.CP_R8, decoder.CP_HL, decoder.CP_N:
		return bit8OpCompare
	case decoder.INC_R8, decoder.INC_HL:
		return bit8OpIncrease
	case decoder.DEC_R8, decoder.DEC_HL:
		return bit8OpDecrease
	case decoder.AND_R8, decoder.AND_HL, decoder.AND_N:
		return bit8OpAnd
	case decoder.OR_R8, decoder.OR_HL, decoder.OR_N:
		return bit8OpOr
	case decoder.XOR_R8, decoder.XOR_HL, decoder.XOR_N:
		return bit8OpXor
	}

	panic(fmt.Sprintf("%d is not a 8-bit arithmetic opcode", instr.InstructionType))
}
