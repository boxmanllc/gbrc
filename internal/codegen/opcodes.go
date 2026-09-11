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

func (cg *Codegen) nop(instr *decoder.Instruction) (*ir.Func, error) {
	return cg.buildVoidFunc(instr, func(b *ir.Block) error { return nil })
}

func (cg *Codegen) ld_r8_r8(instr *decoder.Instruction) (*ir.Func, error) {
	return cg.buildVoidFunc(instr, func(b *ir.Block) error {
		src, err := cg.findReg8GlobalDef(instr.Reg8Src)
		if err != nil {
			return err
		}

		dest, err := cg.findReg8GlobalDef(instr.Reg8Dest)
		if err != nil {
			return err
		}

		srcVal := b.NewLoad(types.I8, src)
		b.NewStore(srcVal, dest)
		return nil
	})
}

func (cg *Codegen) ld_r8_n(instr *decoder.Instruction) (*ir.Func, error) {
	return cg.buildParamFunc(instr, ir.NewParam("n", types.I8), func(b *ir.Block, p *ir.Param) error {
		dest, err := cg.findReg8GlobalDef(instr.Reg8Dest)
		if err != nil {
			return err
		}

		b.NewStore(p, dest)
		return nil
	})
}

func (cg *Codegen) ld_r8_hl(instr *decoder.Instruction) (*ir.Func, error) {
	return cg.buildVoidFunc(instr, func(b *ir.Block) error {
		hl, err := cg.findReg16GlobalDefs(b, decoder.Reg16HL)
		if err != nil {
			return err
		}

		dest, err := cg.findReg8GlobalDef(instr.Reg8Dest)
		if err != nil {
			return err
		}

		hlVal := cg.readReg16(b, hl)
		b.NewStore(cg.readMemory(b, hlVal), dest)
		return nil
	})
}

func (cg *Codegen) ld_hl_r8(instr *decoder.Instruction) (*ir.Func, error) {
	return cg.buildVoidFunc(instr, func(b *ir.Block) error {
		hl, err := cg.findReg16GlobalDefs(b, decoder.Reg16HL)
		if err != nil {
			return err
		}

		src, err := cg.findReg8GlobalDef(instr.Reg8Src)
		if err != nil {
			return err
		}

		hlVal := cg.readReg16(b, hl)
		srcVal := b.NewLoad(types.I8, src)
		cg.updateMemory(b, hlVal, srcVal)
		return nil
	})
}

func (cg *Codegen) ld_hl_n(instr *decoder.Instruction) (*ir.Func, error) {
	return cg.buildParamFunc(instr, ir.NewParam("n", types.I8), func(b *ir.Block, p *ir.Param) error {
		hlReg, err := cg.findReg16GlobalDefs(b, decoder.Reg16HL)
		if err != nil {
			return err
		}

		hl := cg.readReg16(b, hlReg)
		cg.updateMemory(b, hl, p)
		return nil
	})
}

func (cg *Codegen) ld_a_bc(instr *decoder.Instruction) (*ir.Func, error) {
	return cg.buildVoidFunc(instr, func(b *ir.Block) error {
		bc, err := cg.findReg16GlobalDefs(b, decoder.Reg16BC)
		if err != nil {
			return err
		}

		bcVal := cg.readReg16(b, bc)
		b.NewStore(cg.readMemory(b, bcVal), cg.aReg)
		return nil
	})
}

func (cg *Codegen) ld_a_de(instr *decoder.Instruction) (*ir.Func, error) {
	return cg.buildVoidFunc(instr, func(b *ir.Block) error {
		de, err := cg.findReg16GlobalDefs(b, decoder.Reg16DE)
		if err != nil {
			return err
		}

		deVal := cg.readReg16(b, de)
		b.NewStore(cg.readMemory(b, deVal), cg.aReg)
		return nil
	})
}

func (cg *Codegen) ld_bc_a(instr *decoder.Instruction) (*ir.Func, error) {
	return cg.buildVoidFunc(instr, func(b *ir.Block) error {
		bc, err := cg.findReg16GlobalDefs(b, decoder.Reg16BC)
		if err != nil {
			return err
		}

		bcVal := cg.readReg16(b, bc)
		a := b.NewLoad(types.I8, cg.aReg)
		cg.updateMemory(b, bcVal, a)
		return nil
	})
}

func (cg *Codegen) ld_de_a(instr *decoder.Instruction) (*ir.Func, error) {
	return cg.buildVoidFunc(instr, func(b *ir.Block) error {
		de, err := cg.findReg16GlobalDefs(b, decoder.Reg16DE)
		if err != nil {
			return err
		}

		deVal := cg.readReg16(b, de)
		a := b.NewLoad(types.I8, cg.aReg)
		cg.updateMemory(b, deVal, a)
		return nil
	})
}

func (cg *Codegen) ld_a_nn(instr *decoder.Instruction) (*ir.Func, error) {
	return cg.buildParamFunc(instr, ir.NewParam("nn", types.I16), func(b *ir.Block, p *ir.Param) error {
		b.NewStore(cg.readMemory(b, p), cg.aReg)
		return nil
	})
}

func (cg *Codegen) ld_nn_a(instr *decoder.Instruction) (*ir.Func, error) {
	return cg.buildParamFunc(instr, ir.NewParam("nn", types.I16), func(b *ir.Block, p *ir.Param) error {
		a := b.NewLoad(types.I8, cg.aReg)
		cg.updateMemory(b, p, a)
		return nil
	})
}

func (cg *Codegen) ldh_a_c(instr *decoder.Instruction) (*ir.Func, error) {
	return cg.buildVoidFunc(instr, func(b *ir.Block) error {
		addr := cg.calculateHighPageAddress(b, b.NewLoad(types.I8, cg.cReg))
		b.NewStore(cg.readMemory(b, addr), cg.aReg)
		return nil
	})
}

func (cg *Codegen) ldh_c_a(instr *decoder.Instruction) (*ir.Func, error) {
	return cg.buildVoidFunc(instr, func(b *ir.Block) error {
		addr := cg.calculateHighPageAddress(b, b.NewLoad(types.I8, cg.cReg))
		a := b.NewLoad(types.I8, cg.aReg)
		cg.updateMemory(b, addr, a)
		return nil
	})
}

func (cg *Codegen) ldh_a_n(instr *decoder.Instruction) (*ir.Func, error) {
	return cg.buildParamFunc(instr, ir.NewParam("n", types.I8), func(b *ir.Block, p *ir.Param) error {
		addr := cg.calculateHighPageAddress(b, p)
		b.NewStore(cg.readMemory(b, addr), cg.aReg)
		return nil
	})
}

func (cg *Codegen) ldh_n_a(instr *decoder.Instruction) (*ir.Func, error) {
	return cg.buildParamFunc(instr, ir.NewParam("n", types.I8), func(b *ir.Block, p *ir.Param) error {
		addr := cg.calculateHighPageAddress(b, p)
		a := b.NewLoad(types.I8, cg.aReg)
		cg.updateMemory(b, addr, a)
		return nil
	})
}

func (cg *Codegen) ld_a_hl_dec(instr *decoder.Instruction) (*ir.Func, error) {
	return cg.buildVoidFunc(instr, func(b *ir.Block) error {
		hl, err := cg.findReg16GlobalDefs(b, decoder.Reg16HL)
		if err != nil {
			return err
		}

		hlVal := cg.readReg16(b, hl)
		b.NewStore(cg.readMemory(b, hlVal), cg.aReg)

		hlDec := b.NewSub(hlVal, constant.NewInt(types.I16, 1))
		cg.updateReg16(b, hl, hlDec)
		return nil
	})
}

func (cg *Codegen) ld_hl_dec_a(instr *decoder.Instruction) (*ir.Func, error) {
	return cg.buildVoidFunc(instr, func(b *ir.Block) error {
		hl, err := cg.findReg16GlobalDefs(b, decoder.Reg16HL)
		if err != nil {
			return err
		}

		hlVal := cg.readReg16(b, hl)
		a := b.NewLoad(types.I8, cg.aReg)
		cg.updateMemory(b, hlVal, a)

		hlDec := b.NewSub(hlVal, constant.NewInt(types.I16, 1))
		cg.updateReg16(b, hl, hlDec)
		return nil
	})
}

func (cg *Codegen) ld_a_hl_inc(instr *decoder.Instruction) (*ir.Func, error) {
	return cg.buildVoidFunc(instr, func(b *ir.Block) error {
		hl, err := cg.findReg16GlobalDefs(b, decoder.Reg16HL)
		if err != nil {
			return err
		}

		hlVal := cg.readReg16(b, hl)
		b.NewStore(cg.readMemory(b, hlVal), cg.aReg)

		hlInc := b.NewAdd(hlVal, constant.NewInt(types.I16, 1))
		cg.updateReg16(b, hl, hlInc)
		return nil
	})
}

func (cg *Codegen) ld_hl_inc_a(instr *decoder.Instruction) (*ir.Func, error) {
	return cg.buildVoidFunc(instr, func(b *ir.Block) error {
		hl, err := cg.findReg16GlobalDefs(b, decoder.Reg16HL)
		if err != nil {
			return err
		}

		hlVal := cg.readReg16(b, hl)
		a := b.NewLoad(types.I8, cg.aReg)
		cg.updateMemory(b, hlVal, a)

		hlInc := b.NewAdd(hlVal, constant.NewInt(types.I16, 1))
		cg.updateReg16(b, hl, hlInc)
		return nil
	})
}

func (cg *Codegen) ld_r16_nn(instr *decoder.Instruction) (*ir.Func, error) {
	return cg.buildParamFunc(instr, ir.NewParam("nn", types.I16), func(b *ir.Block, p *ir.Param) error {
		r16, err := cg.findReg16GlobalDefs(b, instr.Reg16)
		if err != nil {
			return err
		}

		cg.updateReg16(b, r16, p)
		return nil
	})
}

func (cg *Codegen) ld_nn_sp(instr *decoder.Instruction) (*ir.Func, error) {
	return cg.buildParamFunc(instr, ir.NewParam("nn", types.I16), func(b *ir.Block, p *ir.Param) error {
		sp, err := cg.findReg16GlobalDefs(b, decoder.Reg16SP)
		if err != nil {
			return err
		}

		spVal := cg.readReg16(b, sp)
		lsb := b.NewTrunc(spVal, types.I8)
		msb := b.NewTrunc(b.NewLShr(spVal, constant.NewInt(types.I16, 8)), types.I8)

		nextAddr := b.NewAdd(p, constant.NewInt(types.I16, 1))

		cg.updateMemory(b, p, lsb)
		cg.updateMemory(b, nextAddr, msb)
		return nil
	})
}

func (cg *Codegen) ld_sp_hl(instr *decoder.Instruction) (*ir.Func, error) {
	return cg.buildVoidFunc(instr, func(b *ir.Block) error {
		sp, err := cg.findReg16GlobalDefs(b, decoder.Reg16SP)
		if err != nil {
			return err
		}

		hl, err := cg.findReg16GlobalDefs(b, decoder.Reg16HL)
		if err != nil {
			return err
		}

		hlVal := cg.readReg16(b, hl)
		cg.updateReg16(b, sp, hlVal)
		return nil
	})
}

func (cg *Codegen) ld_hl_sp_e(instr *decoder.Instruction) (*ir.Func, error) {
	return cg.buildParamFunc(instr, ir.NewParam("e", types.I8), func(b *ir.Block, p *ir.Param) error {
		sp, err := cg.findReg16GlobalDefs(b, decoder.Reg16SP)
		if err != nil {
			return err
		}

		hl, err := cg.findReg16GlobalDefs(b, decoder.Reg16HL)
		if err != nil {
			return err
		}

		spVal := cg.readReg16(b, sp)

		eSigned16 := b.NewSExt(p, types.I16)
		eUnsigned16 := b.NewZExt(p, types.I16)

		result := b.NewAdd(spVal, eSigned16)
		hFlag, cFlag := cg.calculateOffsetFlags(b, spVal, eUnsigned16)

		cg.updateReg16(b, hl, result)
		b.NewStore(constant.NewInt(types.I1, 0), cg.zFlag)
		b.NewStore(constant.NewInt(types.I1, 0), cg.nFlag)
		b.NewStore(hFlag, cg.hFlag)
		b.NewStore(cFlag, cg.cFlag)
		return nil
	})
}

func (cg *Codegen) push_r16(instr *decoder.Instruction) (*ir.Func, error) {
	return cg.buildVoidFunc(instr, func(b *ir.Block) error {
		r16, err := cg.findReg16GlobalDefs(b, instr.Reg16)
		if err != nil {
			return err
		}

		sp, err := cg.findReg16GlobalDefs(b, decoder.Reg16SP)
		if err != nil {
			return err
		}

		spVal := cg.readReg16(b, sp)

		spVal = b.NewSub(spVal, constant.NewInt(types.I16, 1))
		cg.updateReg16(b, sp, spVal)

		msbVal := b.NewLoad(types.I8, r16.msb)
		cg.updateMemory(b, spVal, msbVal)

		spVal = b.NewSub(spVal, constant.NewInt(types.I16, 1))
		cg.updateReg16(b, sp, spVal)

		lsbVal := b.NewLoad(types.I8, r16.lsb)
		cg.updateMemory(b, spVal, lsbVal)
		return nil
	})
}

func (cg *Codegen) pop_r16(instr *decoder.Instruction) (*ir.Func, error) {
	return cg.buildVoidFunc(instr, func(b *ir.Block) error {
		r16, err := cg.findReg16GlobalDefs(b, instr.Reg16)
		if err != nil {
			return err
		}

		sp, err := cg.findReg16GlobalDefs(b, decoder.Reg16SP)
		if err != nil {
			return err
		}

		spVal := cg.readReg16(b, sp)

		lsb := cg.readMemory(b, spVal)
		spVal = b.NewAdd(spVal, constant.NewInt(types.I16, 1))
		cg.updateReg16(b, sp, spVal)

		msb := cg.readMemory(b, spVal)
		spVal = b.NewAdd(spVal, constant.NewInt(types.I16, 1))
		cg.updateReg16(b, sp, spVal)

		lsb16 := b.NewZExt(lsb, types.I16)
		msb16 := b.NewZExt(msb, types.I16)
		msbShifted := b.NewShl(msb16, constant.NewInt(types.I16, 8))
		newVal := b.NewOr(msbShifted, lsb16)

		cg.updateReg16(b, r16, newVal)
		return nil
	})
}

func (cg *Codegen) bit8_arithmetic_r8(instr *decoder.Instruction) (*ir.Func, error) {
	return cg.buildVoidFunc(instr, func(b *ir.Block) error {
		srcReg, err := cg.findReg8GlobalDef(instr.Reg8Src)
		if err != nil {
			return err
		}

		operand := b.NewLoad(types.I8, srcReg)
		opType, err := cg.bit8ArithmeticInstrTypeToOpType(instr)
		if err != nil {
			return err
		}

		dest := cg.aReg
		if instr.InstructionType == decoder.INC_R8 || instr.InstructionType == decoder.DEC_R8 {
			dest = srcReg
		}

		toIncludeCarryFlag := false
		if instr.InstructionType == decoder.ADC_R8 || instr.InstructionType == decoder.SBC_R8 {
			toIncludeCarryFlag = true
		}

		if err := cg.perform8BitArithmetic(b, opType, operand, bit8ArithmeticConfig{
			toIncludeCarryFlag: toIncludeCarryFlag,
			destConfig: destConfig{
				destType:     destReg,
				destLocation: dest,
			},
		}); err != nil {
			return err
		}

		return nil
	})
}

func (cg *Codegen) bit8_arithmetic_hl(instr *decoder.Instruction) (*ir.Func, error) {
	return cg.buildVoidFunc(instr, func(b *ir.Block) error {
		hlReg, err := cg.findReg16GlobalDefs(b, decoder.Reg16HL)
		if err != nil {
			return err
		}

		hl := cg.readReg16(b, hlReg)
		operand := cg.readMemory(b, hl)
		opType, err := cg.bit8ArithmeticInstrTypeToOpType(instr)
		if err != nil {
			return err
		}

		dest := value.Value(cg.aReg)
		destType := destReg
		if instr.InstructionType == decoder.INC_HL || instr.InstructionType == decoder.DEC_HL {
			dest = hl
			destType = destHL
		}

		toIncludeCarryFlag := false
		if instr.InstructionType == decoder.ADC_HL || instr.InstructionType == decoder.SBC_HL {
			toIncludeCarryFlag = true
		}

		if err := cg.perform8BitArithmetic(b, opType, operand, bit8ArithmeticConfig{
			toIncludeCarryFlag: toIncludeCarryFlag,
			destConfig: destConfig{
				destType:     destType,
				destLocation: dest,
			},
		}); err != nil {
			return err
		}

		return nil
	})
}

func (cg *Codegen) bit8_arithmetic_n(instr *decoder.Instruction) (*ir.Func, error) {
	return cg.buildParamFunc(instr, ir.NewParam("n", types.I8), func(b *ir.Block, p *ir.Param) error {
		opType, err := cg.bit8ArithmeticInstrTypeToOpType(instr)
		if err != nil {
			return err
		}

		toIncludeCarryFlag := false
		if instr.InstructionType == decoder.ADC_N || instr.InstructionType == decoder.SBC_N {
			toIncludeCarryFlag = true
		}

		if err := cg.perform8BitArithmetic(b, opType, p, bit8ArithmeticConfig{
			toIncludeCarryFlag: toIncludeCarryFlag,
			destConfig: destConfig{
				destType:     destReg,
				destLocation: cg.aReg,
			},
		}); err != nil {
			return err
		}
		return nil
	})
}

func (cg *Codegen) ccf(instr *decoder.Instruction) (*ir.Func, error) {
	return cg.buildVoidFunc(instr, func(b *ir.Block) error {
		cVal := b.NewLoad(types.I1, cg.cFlag)
		flipC := b.NewXor(cVal, constant.NewInt(types.I1, -1))

		b.NewStore(constant.NewInt(types.I1, 0), cg.nFlag)
		b.NewStore(constant.NewInt(types.I1, 0), cg.hFlag)
		b.NewStore(flipC, cg.cFlag)
		return nil
	})
}

func (cg *Codegen) scf(instr *decoder.Instruction) (*ir.Func, error) {
	return cg.buildVoidFunc(instr, func(b *ir.Block) error {
		b.NewStore(constant.NewInt(types.I1, 0), cg.nFlag)
		b.NewStore(constant.NewInt(types.I1, 0), cg.hFlag)
		b.NewStore(constant.NewInt(types.I1, 1), cg.cFlag)
		return nil
	})
}

func (cg *Codegen) cpl(instr *decoder.Instruction) (*ir.Func, error) {
	return cg.buildVoidFunc(instr, func(b *ir.Block) error {
		aVal := b.NewLoad(types.I8, cg.aReg)
		flipA := b.NewXor(aVal, constant.NewInt(types.I8, -1))

		b.NewStore(flipA, cg.aReg)
		b.NewStore(constant.NewInt(types.I1, 1), cg.nFlag)
		b.NewStore(constant.NewInt(types.I1, 1), cg.hFlag)
		return nil
	})
}

func (cg *Codegen) inc_r16(instr *decoder.Instruction) (*ir.Func, error) {
	return cg.buildVoidFunc(instr, func(b *ir.Block) error {
		r16, err := cg.findReg16GlobalDefs(b, instr.Reg16)
		if err != nil {
			return err
		}

		val := cg.readReg16(b, r16)
		val = b.NewAdd(val, constant.NewInt(types.I16, 1))
		cg.updateReg16(b, r16, val)
		return nil
	})
}

func (cg *Codegen) dec_r16(instr *decoder.Instruction) (*ir.Func, error) {
	return cg.buildVoidFunc(instr, func(b *ir.Block) error {
		r16, err := cg.findReg16GlobalDefs(b, instr.Reg16)
		if err != nil {
			return err
		}

		val := cg.readReg16(b, r16)
		val = b.NewSub(val, constant.NewInt(types.I16, 1))
		cg.updateReg16(b, r16, val)
		return nil
	})
}

func (cg *Codegen) add_hl_r16(instr *decoder.Instruction) (*ir.Func, error) {
	return cg.buildVoidFunc(instr, func(b *ir.Block) error {
		hl, err := cg.findReg16GlobalDefs(b, decoder.Reg16HL)
		if err != nil {
			return err
		}

		reg, err := cg.findReg16GlobalDefs(b, instr.Reg16)
		if err != nil {
			return err
		}

		hlVal := cg.readReg16(b, hl)
		regVal := cg.readReg16(b, reg)

		hl32 := b.NewZExt(hlVal, types.I32)
		reg32 := b.NewZExt(regVal, types.I32)

		hlLow := b.NewAnd(hl32, constant.NewInt(types.I32, 0x0FFF))
		regLow := b.NewAnd(reg32, constant.NewInt(types.I32, 0x0FFF))

		sum32 := b.NewAdd(hl32, reg32)
		lowSum := b.NewAdd(hlLow, regLow)

		result := b.NewTrunc(sum32, types.I16)
		hFlag := b.NewICmp(enum.IPredUGE, lowSum, constant.NewInt(types.I32, 0x1000))
		cFlag := b.NewICmp(enum.IPredUGE, sum32, constant.NewInt(types.I32, 0x10000))

		cg.updateReg16(b, hl, result)
		b.NewStore(constant.NewInt(types.I1, 0), cg.nFlag)
		b.NewStore(hFlag, cg.hFlag)
		b.NewStore(cFlag, cg.cFlag)
		return nil
	})
}

func (cg *Codegen) add_sp_e(instr *decoder.Instruction) (*ir.Func, error) {
	return cg.buildParamFunc(instr, ir.NewParam("e", types.I8), func(b *ir.Block, p *ir.Param) error {
		spReg, err := cg.findReg16GlobalDefs(b, decoder.Reg16SP)
		if err != nil {
			return err
		}

		spVal := cg.readReg16(b, spReg)

		eSigned16 := b.NewSExt(p, types.I16)
		eUnsigned16 := b.NewZExt(p, types.I16)

		result := b.NewAdd(spVal, eSigned16)
		hFlag, cFlag := cg.calculateOffsetFlags(b, spVal, eUnsigned16)

		cg.updateReg16(b, spReg, result)
		b.NewStore(constant.NewInt(types.I1, 0), cg.zFlag)
		b.NewStore(constant.NewInt(types.I1, 0), cg.nFlag)
		b.NewStore(hFlag, cg.hFlag)
		b.NewStore(cFlag, cg.cFlag)
		return nil
	})
}

func (cg *Codegen) bitwise(instr *decoder.Instruction) (*ir.Func, error) {
	return cg.buildVoidFunc(instr, func(b *ir.Block) error {
		operand, destType, destLocation, err := cg.resolveOperandAndDest(b, instr)
		if err != nil {
			return err
		}

		opType, err := cg.bitwiseOpFromInstrType(instr)
		if err != nil {
			return err
		}

		cg.performBitwise(b, opType, operand, bitwiseConfig{
			updateZeroFlag: instr.IsCbPrefixed,
			destConfig: destConfig{
				destType:     destType,
				destLocation: destLocation,
			},
		})
		return nil
	})
}

func (cg *Codegen) bit_op(instr *decoder.Instruction) (*ir.Func, error) {
	return cg.buildParamFunc(instr, ir.NewParam("n", types.I8), func(b *ir.Block, p *ir.Param) error {
		operand, destType, destLocation, err := cg.resolveOperandAndDest(b, instr)
		if err != nil {
			return err
		}

		opType, err := cg.bitOpFromInstrType(instr)
		if err != nil {
			return err
		}

		cg.performBitOp(b, opType, p, operand, destConfig{
			destType:     destType,
			destLocation: destLocation,
		})

		return nil
	})
}

func (cg *Codegen) jp_nn(instr *decoder.Instruction, irBlock *ir.Block) error {
	cg.increaseCycles(instr, irBlock)
	toBlock, ok := cg.irBlocks[instr.Imm16Bit]
	if !ok {
		return fmt.Errorf("cannot find jp nn destination block")
	}

	irBlock.NewBr(toBlock)
	return nil
}
