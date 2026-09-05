package codegen

import (
	"fmt"
	"regexp"

	"github.com/0xmukesh/boxman/internal/decoder"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/enum"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

type bit8ArithmeticOp int
type bit8ArithmeticDest int

type reg16Store struct {
	msb, lsb   value.Value // value of two 8-bit registers which are composed together to create 16-bit register
	val        value.Value // raw value of 16-bit register
	isSplitUp  bool        // whether register's value composes of a single 16-bit register or combination of two 8-bit register
	hasFlagReg bool        // whether F register is involved
}

type bit8ArithemticConfig struct {
	destType           bit8ArithmeticDest
	destLocation       value.Value
	toIncludeCarryFlag bool
}

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
	bit8DestReg bit8ArithmeticDest = iota // store result in a register
	bit8DestHL                            // store result in location pointed by (HL)
)

var (
	mnemonicPat = regexp.MustCompile(`[ ,()]`)
)

func mnemonicToFuncName(mnemonic string) string {
	return mnemonicPat.ReplaceAllString(mnemonic, "_")
}

func (cg *Codegen) getRamPtr(irBlock *ir.Block, addr value.Value) value.Value {
	ramp := irBlock.NewBitCast(cg.ram, types.NewPointer(types.I8))
	idx := irBlock.NewZExt(addr, types.I64)
	return irBlock.NewGetElementPtr(types.I8, ramp, idx)
}

func (cg *Codegen) readMemory(irBlock *ir.Block, addr value.Value) value.Value {
	return irBlock.NewLoad(types.I8, cg.getRamPtr(irBlock, addr))
}

func (cg *Codegen) updateMemory(irBlock *ir.Block, addr value.Value, val value.Value) {
	irBlock.NewStore(val, cg.getRamPtr(irBlock, addr))
}

func (cg *Codegen) findReg8GlobalDef(reg8 decoder.Reg8) *ir.Global {
	switch reg8 {
	case decoder.Reg8A:
		return cg.aReg
	case decoder.Reg8B:
		return cg.bReg
	case decoder.Reg8C:
		return cg.cReg
	case decoder.Reg8D:
		return cg.dReg
	case decoder.Reg8E:
		return cg.eReg
	case decoder.Reg8H:
		return cg.hReg
	case decoder.Reg8L:
		return cg.lReg
	default:
		panic(fmt.Sprintf("cannot find llvm global def for %d reg8 type", reg8))
	}
}

func (cg *Codegen) findReg16GlobalDefs(irBlock *ir.Block, reg16 decoder.Reg16) reg16Store {
	switch reg16 {
	case decoder.Reg16BC:
		return reg16Store{
			isSplitUp: true,
			msb:       cg.bReg,
			lsb:       cg.cReg,
		}
	case decoder.Reg16DE:
		return reg16Store{
			isSplitUp: true,
			msb:       cg.dReg,
			lsb:       cg.eReg,
		}
	case decoder.Reg16HL:
		return reg16Store{
			isSplitUp: true,
			msb:       cg.hReg,
			lsb:       cg.lReg,
		}
	case decoder.Reg16AF:
		return reg16Store{
			isSplitUp:  true,
			hasFlagReg: true,
			msb:        cg.aReg,
			lsb:        cg.buildFReg(irBlock),
		}
	case decoder.Reg16SP:
		return reg16Store{
			isSplitUp: false,
			val:       cg.sp,
		}
	default:
		panic(fmt.Sprintf("cannot find llvm global def for %d reg16 type", reg16))
	}
}

func (cg *Codegen) readReg16(irBlock *ir.Block, reg16 reg16Store) value.Value {
	if reg16.isSplitUp {
		msbVal := irBlock.NewLoad(types.I8, reg16.msb)
		var lsb16 *ir.InstZExt

		if reg16.hasFlagReg {
			// F register is built on the fly so additional load instruction isn't required
			lsb16 = irBlock.NewZExt(reg16.lsb, types.I16)
		} else {
			lsbVal := irBlock.NewLoad(types.I8, reg16.lsb)
			lsb16 = irBlock.NewZExt(lsbVal, types.I16)
		}

		msb16 := irBlock.NewZExt(msbVal, types.I16)
		msbShifted := irBlock.NewShl(msb16, constant.NewInt(types.I16, 8))
		return irBlock.NewOr(msbShifted, lsb16)
	} else {
		return irBlock.NewLoad(types.I16, reg16.val)
	}
}

func (cg *Codegen) updateReg16(irBlock *ir.Block, reg16 reg16Store, newVal value.Value) {
	if reg16.isSplitUp {
		msb16 := irBlock.NewLShr(newVal, constant.NewInt(types.I16, 8))
		msbVal := irBlock.NewTrunc(msb16, types.I8)
		lsbVal := irBlock.NewTrunc(newVal, types.I8)

		irBlock.NewStore(msbVal, reg16.msb)

		if reg16.hasFlagReg {
			// F register is not a single global register so this would update all the different global registers for individual flags
			cg.updateFReg(irBlock, lsbVal)
		} else {
			irBlock.NewStore(lsbVal, reg16.lsb)
		}
	} else {
		irBlock.NewStore(newVal, reg16.val)
	}
}

func (cg *Codegen) buildFReg(irBlock *ir.Block) value.Value {
	z := irBlock.NewLoad(types.I1, cg.zFlag)
	n := irBlock.NewLoad(types.I1, cg.nFlag)
	h := irBlock.NewLoad(types.I1, cg.hFlag)
	c := irBlock.NewLoad(types.I1, cg.cFlag)

	z8 := irBlock.NewSelect(
		z,
		constant.NewInt(types.I8, 0x80),
		constant.NewInt(types.I8, 0x00),
	)
	n8 := irBlock.NewSelect(
		n,
		constant.NewInt(types.I8, 0x40),
		constant.NewInt(types.I8, 0x00),
	)
	h8 := irBlock.NewSelect(
		h,
		constant.NewInt(types.I8, 0x20),
		constant.NewInt(types.I8, 0x00),
	)
	c8 := irBlock.NewSelect(
		c,
		constant.NewInt(types.I8, 0x10),
		constant.NewInt(types.I8, 0x00),
	)

	f := irBlock.NewOr(z8, n8)
	f = irBlock.NewOr(f, h8)
	f = irBlock.NewOr(f, c8)

	return f
}

func (cg *Codegen) updateFReg(irBlock *ir.Block, val value.Value) {
	z := irBlock.NewTrunc(
		irBlock.NewLShr(val, constant.NewInt(types.I8, 7)),
		types.I1,
	)
	n := irBlock.NewTrunc(
		irBlock.NewLShr(val, constant.NewInt(types.I8, 6)),
		types.I1,
	)
	h := irBlock.NewTrunc(
		irBlock.NewLShr(val, constant.NewInt(types.I8, 5)),
		types.I1,
	)
	c := irBlock.NewTrunc(
		irBlock.NewLShr(val, constant.NewInt(types.I8, 4)),
		types.I1,
	)

	irBlock.NewStore(z, cg.zFlag)
	irBlock.NewStore(n, cg.nFlag)
	irBlock.NewStore(h, cg.hFlag)
	irBlock.NewStore(c, cg.cFlag)
}

func (cg *Codegen) perform8BitArithmetic(
	irBlock *ir.Block, opType bit8ArithmeticOp,
	operand value.Value, cfg bit8ArithemticConfig,
) {
	var a, aVal, a16, operand16, c16 value.Value
	var result, nFlag, hFlag, cFlag value.Value

	if opType != bit8OpIncrease && opType != bit8OpDecrease {
		a = cg.findReg8GlobalDef(decoder.Reg8A)
		aVal = irBlock.NewLoad(types.I8, a)
	}

	if opType == bit8OpAdd || opType == bit8OpSub || opType == bit8OpCompare {
		a16 = irBlock.NewZExt(aVal, types.I16)
		operand16 = irBlock.NewZExt(operand, types.I16)
	}

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
		nFlag = constant.NewInt(types.I8, 0)
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
		nFlag = constant.NewInt(types.I8, 1)
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

		nFlag = constant.NewInt(types.I8, 0)
		hFlag = irBlock.NewICmp(enum.IPredUGE, sumLow, constant.NewInt(types.I8, 0x10))
	case bit8OpDecrease:
		// stores result in either source register or location point by (HL)
		// flags:
		//   z: result == 0
		//   n: 1
		//   h: (a & 0x0F) < 1
		result = irBlock.NewSub(a, constant.NewInt(types.I8, 1))

		aLow := irBlock.NewAnd(a, constant.NewInt(types.I8, 0x0F))

		nFlag = constant.NewInt(types.I8, 1)
		hFlag = irBlock.NewICmp(enum.IPredUGT, aLow, constant.NewInt(types.I8, 1))
	case bit8OpAnd:
		result = irBlock.NewAnd(a, operand)
		nFlag = constant.NewInt(types.I8, 0)
		hFlag = constant.NewInt(types.I8, 1)
		cFlag = constant.NewInt(types.I8, 0)
	case bit8OpOr:
		result = irBlock.NewOr(a, operand)
		nFlag = constant.NewInt(types.I8, 0)
		hFlag = constant.NewInt(types.I8, 0)
		cFlag = constant.NewInt(types.I8, 0)
	case bit8OpXor:
		result = irBlock.NewXor(a, operand)
		nFlag = constant.NewInt(types.I8, 0)
		hFlag = constant.NewInt(types.I8, 0)
		cFlag = constant.NewInt(types.I8, 0)
	}

	zFlag := irBlock.NewICmp(enum.IPredEQ, result, constant.NewInt(types.I8, 0))

	if opType != bit8OpCompare {
		switch cfg.destType {
		case bit8DestReg:
			irBlock.NewStore(result, cfg.destLocation)
		case bit8DestHL:
			cg.updateMemory(irBlock, cfg.destLocation, result)
		}
	}

	irBlock.NewStore(zFlag, cg.zFlag)
	irBlock.NewStore(nFlag, cg.nFlag)
	irBlock.NewStore(hFlag, cg.hFlag)

	if cFlag != nil {
		irBlock.NewStore(cFlag, cg.cFlag)
	}
}

func (cg *Codegen) increaseCycles(instr *decoder.Instruction, irBlock *ir.Block) {
	cycles := irBlock.NewLoad(types.I32, cg.cycles)
	cyclesInc := irBlock.NewAdd(cycles, constant.NewInt(types.I32, int64(instr.BaseMCycles)))
	irBlock.NewStore(cyclesInc, cg.cycles)
}

func (cg *Codegen) buildVoidFunc(instr *decoder.Instruction, build func(*ir.Block)) *ir.Func {
	fn := cg.module.NewFunc(mnemonicToFuncName(instr.Mnemonic), types.Void)
	entry := fn.NewBlock("entry")
	build(entry)
	cg.increaseCycles(instr, entry)
	entry.NewRet(nil)
	return fn
}

func (cg *Codegen) buildParamFunc(instr *decoder.Instruction, param *ir.Param, build func(*ir.Block, *ir.Param)) *ir.Func {
	fn := cg.module.NewFunc(mnemonicToFuncName(instr.Mnemonic), types.Void, param)
	entry := fn.NewBlock("entry")
	build(entry, param)
	cg.increaseCycles(instr, entry)
	entry.NewRet(nil)
	return fn
}

func (cg *Codegen) calculateHighPageAddress(irBlock *ir.Block, val value.Value) value.Value {
	val16 := irBlock.NewZExt(val, types.I16)
	return irBlock.NewAdd(constant.NewInt(types.I16, 0xFF00), val16)
}

func (cg *Codegen) setupDebugFunc() {
	printfFunc := cg.module.NewFunc("printf", types.I32, ir.NewParam("", types.NewPointer(types.I8)))
	printfFunc.Sig.Variadic = true

	fmtStr := constant.NewCharArrayFromString("A=%02X B=%02X C=%02X D=%02X E=%02X H=%02X L=%02X F=%c%c%c%c cycles=%d\n\x00")
	fmtGlobal := cg.module.NewGlobalDef("fmt", fmtStr)
	fmtGlobal.Linkage = enum.LinkagePrivate
	fmtGlobal.Immutable = true

	debugFunc := cg.module.NewFunc("debug", types.Void)
	cg.debugFunc = debugFunc
	entry := debugFunc.NewBlock("entry")

	aReg := entry.NewLoad(types.I8, cg.aReg)
	bReg := entry.NewLoad(types.I8, cg.bReg)
	cReg := entry.NewLoad(types.I8, cg.cReg)
	dReg := entry.NewLoad(types.I8, cg.dReg)
	eReg := entry.NewLoad(types.I8, cg.eReg)
	hReg := entry.NewLoad(types.I8, cg.hReg)
	lReg := entry.NewLoad(types.I8, cg.lReg)
	zFlag := entry.NewLoad(types.I1, cg.zFlag)
	nFlag := entry.NewLoad(types.I1, cg.nFlag)
	hFlag := entry.NewLoad(types.I1, cg.hFlag)
	cFlag := entry.NewLoad(types.I1, cg.cFlag)
	cycles := entry.NewLoad(types.I32, cg.cycles)

	zChar := entry.NewSelect(
		zFlag,
		constant.NewInt(types.I8, 'Z'),
		constant.NewInt(types.I8, '-'),
	)
	nChar := entry.NewSelect(
		nFlag,
		constant.NewInt(types.I8, 'N'),
		constant.NewInt(types.I8, '-'),
	)
	hChar := entry.NewSelect(
		hFlag,
		constant.NewInt(types.I8, 'H'),
		constant.NewInt(types.I8, '-'),
	)
	cChar := entry.NewSelect(
		cFlag,
		constant.NewInt(types.I8, 'C'),
		constant.NewInt(types.I8, '-'),
	)

	zChar32 := entry.NewZExt(zChar, types.I32)
	nChar32 := entry.NewZExt(nChar, types.I32)
	hChar32 := entry.NewZExt(hChar, types.I32)
	cChar32 := entry.NewZExt(cChar, types.I32)

	a32 := entry.NewZExt(aReg, types.I32)
	b32 := entry.NewZExt(bReg, types.I32)
	c32 := entry.NewZExt(cReg, types.I32)
	d32 := entry.NewZExt(dReg, types.I32)
	e32 := entry.NewZExt(eReg, types.I32)
	h32 := entry.NewZExt(hReg, types.I32)
	l32 := entry.NewZExt(lReg, types.I32)

	zero := constant.NewInt(types.I32, 0)
	fmtPtr := entry.NewGetElementPtr(fmtStr.Type(), fmtGlobal, zero, zero)

	entry.NewCall(printfFunc, fmtPtr,
		a32, b32, c32, d32, e32, h32, l32,
		zChar32, nChar32, hChar32, cChar32, cycles,
	)
	entry.NewRet(nil)
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
