package codegen

import (
	"fmt"

	"github.com/0xmukesh/boxman/internal/decoder"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

func (cg *Codegen) nop(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {})
}

func (cg *Codegen) ld_r8_r8(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		src := b.NewLoad(types.I8, cg.findReg8GlobalDef(instr.Reg8Src))
		b.NewStore(src, cg.findReg8GlobalDef(instr.Reg8Dest))
	})
}

func (cg *Codegen) ld_r8_n(instr *decoder.Instruction) *ir.Func {
	return cg.buildParamFunc(instr, ir.NewParam("n", types.I8), func(b *ir.Block, p *ir.Param) {
		b.NewStore(p, cg.findReg8GlobalDef(instr.Reg8Dest))
	})
}

func (cg *Codegen) ld_r8_hl(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		hl := cg.readReg16(b, cg.findReg16GlobalDefs(b, decoder.Reg16HL))
		b.NewStore(cg.readMemory(b, hl), cg.findReg8GlobalDef(instr.Reg8Dest))
	})
}

func (cg *Codegen) ld_hl_r8(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		hl := cg.readReg16(b, cg.findReg16GlobalDefs(b, decoder.Reg16HL))
		src := b.NewLoad(types.I8, cg.findReg8GlobalDef(instr.Reg8Src))
		cg.updateMemory(b, hl, src)
	})
}

func (cg *Codegen) ld_hl_n(instr *decoder.Instruction) *ir.Func {
	return cg.buildParamFunc(instr, ir.NewParam("n", types.I8), func(b *ir.Block, p *ir.Param) {
		hl := cg.readReg16(b, cg.findReg16GlobalDefs(b, decoder.Reg16HL))
		cg.updateMemory(b, hl, p)
	})
}

func (cg *Codegen) ld_a_bc(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		bc := cg.readReg16(b, cg.findReg16GlobalDefs(b, decoder.Reg16BC))
		b.NewStore(cg.readMemory(b, bc), cg.aReg)
	})
}

func (cg *Codegen) ld_a_de(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		de := cg.readReg16(b, cg.findReg16GlobalDefs(b, decoder.Reg16DE))
		b.NewStore(cg.readMemory(b, de), cg.aReg)
	})
}

func (cg *Codegen) ld_bc_a(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		bc := cg.readReg16(b, cg.findReg16GlobalDefs(b, decoder.Reg16BC))
		a := b.NewLoad(types.I8, cg.aReg)
		cg.updateMemory(b, bc, a)
	})
}

func (cg *Codegen) ld_de_a(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		de := cg.readReg16(b, cg.findReg16GlobalDefs(b, decoder.Reg16DE))
		a := b.NewLoad(types.I8, cg.aReg)
		cg.updateMemory(b, de, a)
	})
}

func (cg *Codegen) ld_a_nn(instr *decoder.Instruction) *ir.Func {
	return cg.buildParamFunc(instr, ir.NewParam("nn", types.I16), func(b *ir.Block, p *ir.Param) {
		b.NewStore(cg.readMemory(b, p), cg.aReg)
	})
}

func (cg *Codegen) ld_nn_a(instr *decoder.Instruction) *ir.Func {
	return cg.buildParamFunc(instr, ir.NewParam("nn", types.I16), func(b *ir.Block, p *ir.Param) {
		a := b.NewLoad(types.I8, cg.aReg)
		cg.updateMemory(b, p, a)
	})
}

func (cg *Codegen) ldh_a_c(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		addr := cg.calculateHighPageAddress(b, b.NewLoad(types.I8, cg.cReg))
		b.NewStore(cg.readMemory(b, addr), cg.aReg)
	})
}

func (cg *Codegen) ldh_c_a(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		addr := cg.calculateHighPageAddress(b, b.NewLoad(types.I8, cg.cReg))
		a := b.NewLoad(types.I8, cg.aReg)
		cg.updateMemory(b, addr, a)
	})
}

func (cg *Codegen) ldh_a_n(instr *decoder.Instruction) *ir.Func {
	return cg.buildParamFunc(instr, ir.NewParam("n", types.I8), func(b *ir.Block, p *ir.Param) {
		addr := cg.calculateHighPageAddress(b, p)
		b.NewStore(cg.readMemory(b, addr), cg.aReg)
	})
}

func (cg *Codegen) ldh_n_a(instr *decoder.Instruction) *ir.Func {
	return cg.buildParamFunc(instr, ir.NewParam("n", types.I8), func(b *ir.Block, p *ir.Param) {
		addr := cg.calculateHighPageAddress(b, p)
		a := b.NewLoad(types.I8, cg.aReg)
		cg.updateMemory(b, addr, a)
	})
}

func (cg *Codegen) ld_a_hl_dec(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		r16 := cg.findReg16GlobalDefs(b, decoder.Reg16HL)
		hl := cg.readReg16(b, r16)
		b.NewStore(cg.readMemory(b, hl), cg.aReg)
		hlDec := b.NewSub(hl, constant.NewInt(types.I16, 1))
		cg.updateReg16(b, r16, hlDec)
	})
}

func (cg *Codegen) ld_hl_dec_a(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		r16 := cg.findReg16GlobalDefs(b, decoder.Reg16HL)
		hl := cg.readReg16(b, r16)
		a := b.NewLoad(types.I8, cg.aReg)
		cg.updateMemory(b, hl, a)
		hlDec := b.NewSub(hl, constant.NewInt(types.I16, 1))
		cg.updateReg16(b, r16, hlDec)
	})
}

func (cg *Codegen) ld_a_hl_inc(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		r16 := cg.findReg16GlobalDefs(b, decoder.Reg16HL)
		hl := cg.readReg16(b, r16)
		b.NewStore(cg.readMemory(b, hl), cg.aReg)
		hlInc := b.NewAdd(hl, constant.NewInt(types.I16, 1))
		cg.updateReg16(b, r16, hlInc)
	})
}

func (cg *Codegen) ld_hl_inc_a(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		r16 := cg.findReg16GlobalDefs(b, decoder.Reg16HL)
		hl := cg.readReg16(b, r16)
		a := b.NewLoad(types.I8, cg.aReg)
		cg.updateMemory(b, hl, a)
		hlInc := b.NewAdd(hl, constant.NewInt(types.I16, 1))
		cg.updateReg16(b, r16, hlInc)
	})
}

func (cg *Codegen) ld_r16_nn(instr *decoder.Instruction) *ir.Func {
	return cg.buildParamFunc(instr, ir.NewParam("nn", types.I16), func(b *ir.Block, p *ir.Param) {
		r16 := cg.findReg16GlobalDefs(b, instr.Reg16)
		cg.updateReg16(b, r16, p)
	})
}

func (cg *Codegen) ld_nn_sp(instr *decoder.Instruction) *ir.Func {
	return cg.buildParamFunc(instr, ir.NewParam("nn", types.I16), func(b *ir.Block, p *ir.Param) {
		sp := cg.readReg16(b, cg.findReg16GlobalDefs(b, decoder.Reg16SP))
		lsb := b.NewTrunc(sp, types.I8)
		msb := b.NewTrunc(b.NewLShr(sp, constant.NewInt(types.I16, 8)), types.I8)
		addr := p
		nextAddr := b.NewAdd(p, constant.NewInt(types.I16, 1))
		cg.updateMemory(b, addr, lsb)
		cg.updateMemory(b, nextAddr, msb)
	})
}

func (cg *Codegen) ld_sp_hl(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		sp := cg.findReg16GlobalDefs(b, decoder.Reg16SP)
		hl := cg.readReg16(b, cg.findReg16GlobalDefs(b, decoder.Reg16HL))
		cg.updateReg16(b, sp, hl)
	})
}

func (cg *Codegen) ld_hl_sp_e(instr *decoder.Instruction) *ir.Func {
	return cg.buildParamFunc(instr, ir.NewParam("e", types.I8), func(b *ir.Block, p *ir.Param) {
		sp := cg.readReg16(b, cg.findReg16GlobalDefs(b, decoder.Reg16SP))
		hl := cg.findReg16GlobalDefs(b, decoder.Reg16HL)

		e16 := b.NewSExt(p, types.I16)
		result := b.NewAdd(sp, e16)
		cg.updateReg16(b, hl, result)

		// for finding carry bits:
		// 	sum = lsb of sp + unsigned e
		//  carry bits = (lsb of sp) ^ (unsigned e) ^ sum
		spLow := b.NewTrunc(sp, types.I8)
		spLow16 := b.NewZExt(spLow, types.I16)
		e16u := b.NewZExt(p, types.I16)
		sum16 := b.NewAdd(spLow16, e16u)
		carryPerBit := b.NewXor(b.NewXor(spLow16, e16u), sum16)

		hFlag := b.NewTrunc(b.NewLShr(carryPerBit, constant.NewInt(types.I64, 4)), types.I1)
		cFlag := b.NewTrunc(b.NewLShr(carryPerBit, constant.NewInt(types.I64, 8)), types.I1)

		b.NewStore(constant.NewInt(types.I1, 0), cg.zFlag)
		b.NewStore(constant.NewInt(types.I1, 0), cg.nFlag)
		b.NewStore(hFlag, cg.hFlag)
		b.NewStore(cFlag, cg.cFlag)
	})
}

func (cg *Codegen) push_r16(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		r16 := cg.findReg16GlobalDefs(b, instr.Reg16)
		sp := cg.findReg16GlobalDefs(b, decoder.Reg16SP)
		spVal := cg.readReg16(b, sp)

		spVal = b.NewSub(spVal, constant.NewInt(types.I16, 1))
		cg.updateReg16(b, sp, spVal)
		cg.updateMemory(b, spVal, r16.msb)

		spVal = b.NewSub(spVal, constant.NewInt(types.I16, 1))
		cg.updateReg16(b, sp, spVal)
		cg.updateMemory(b, spVal, r16.lsb)
	})
}

func (cg *Codegen) pop_r16(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		r16 := cg.findReg16GlobalDefs(b, instr.Reg16)
		sp := cg.findReg16GlobalDefs(b, decoder.Reg16SP)
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
	})
}

func (cg *Codegen) bit8_arithmetic_r8(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		src := cg.findReg8GlobalDef(instr.Reg8Src)
		operand := b.NewLoad(types.I8, src)
		opType := cg.bit8ArithmeticInstrTypeToOpType(instr)

		dest := cg.aReg
		if instr.InstructionType == decoder.INC_R8 || instr.InstructionType == decoder.DEC_R8 {
			dest = src
		}

		toIncludeCarryFlag := false
		if instr.InstructionType == decoder.ADC_R8 || instr.InstructionType == decoder.SBC_R8 {
			toIncludeCarryFlag = true
		}

		cg.perform8BitArithmetic(b, opType, operand, bit8ArithemticConfig{
			destType:           bit8DestReg,
			destLocation:       dest,
			toIncludeCarryFlag: toIncludeCarryFlag,
		})
	})
}

func (cg *Codegen) bit8_arithmetic_hl(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		hl := cg.readReg16(b, cg.findReg16GlobalDefs(b, decoder.Reg16HL))
		operand := cg.readMemory(b, hl)
		opType := cg.bit8ArithmeticInstrTypeToOpType(instr)

		dest := value.Value(cg.aReg)
		destType := bit8DestReg
		if instr.InstructionType == decoder.INC_HL || instr.InstructionType == decoder.DEC_HL {
			dest = hl
			destType = bit8DestHL
		}

		toIncludeCarryFlag := false
		if instr.InstructionType == decoder.ADC_HL || instr.InstructionType == decoder.SBC_HL {
			toIncludeCarryFlag = true
		}

		cg.perform8BitArithmetic(b, opType, operand, bit8ArithemticConfig{
			destType:           destType,
			destLocation:       dest,
			toIncludeCarryFlag: toIncludeCarryFlag,
		})
	})
}

func (cg *Codegen) bit8_arithmetic_n(instr *decoder.Instruction) *ir.Func {
	return cg.buildParamFunc(instr, ir.NewParam("n", types.I8), func(b *ir.Block, p *ir.Param) {
		opType := cg.bit8ArithmeticInstrTypeToOpType(instr)

		toIncludeCarryFlag := false
		if instr.InstructionType == decoder.ADC_N || instr.InstructionType == decoder.SBC_N {
			toIncludeCarryFlag = true
		}

		cg.perform8BitArithmetic(b, opType, p, bit8ArithemticConfig{
			destType:           bit8DestReg,
			destLocation:       cg.aReg,
			toIncludeCarryFlag: toIncludeCarryFlag,
		})
	})
}

func (cg *Codegen) ccf(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		cVal := b.NewLoad(types.I1, cg.cFlag)
		flipC := b.NewXor(cVal, constant.NewInt(types.I1, -1))

		b.NewStore(constant.NewInt(types.I8, 0), cg.nFlag)
		b.NewStore(constant.NewInt(types.I8, 0), cg.hFlag)
		b.NewStore(flipC, cg.cFlag)
	})
}

func (cg *Codegen) scf(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		b.NewStore(constant.NewInt(types.I8, 0), cg.nFlag)
		b.NewStore(constant.NewInt(types.I8, 0), cg.hFlag)
		b.NewStore(constant.NewInt(types.I8, 1), cg.cFlag)
	})
}

func (cg *Codegen) cpl(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		aVal := b.NewLoad(types.I8, cg.aReg)
		flipA := b.NewXor(aVal, constant.NewInt(types.I8, -1))

		b.NewStore(flipA, cg.aReg)
		b.NewStore(constant.NewInt(types.I1, 1), cg.nFlag)
		b.NewStore(constant.NewInt(types.I1, 1), cg.hFlag)
	})
}

func (cg *Codegen) inc_r16(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		r16 := cg.findReg16GlobalDefs(b, instr.Reg16)
		val := cg.readReg16(b, r16)
		val = b.NewAdd(val, constant.NewInt(types.I16, 1))
		cg.updateReg16(b, r16, val)
	})
}

func (cg *Codegen) dec_r16(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		r16 := cg.findReg16GlobalDefs(b, instr.Reg16)
		val := cg.readReg16(b, r16)
		val = b.NewSub(val, constant.NewInt(types.I16, 1))
		cg.updateReg16(b, r16, val)
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
