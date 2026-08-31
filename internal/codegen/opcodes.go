package codegen

import (
	"fmt"

	"github.com/0xmukesh/boxman/internal/decoder"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
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
		hl := cg.readReg16(b, cg.findReg16GlobalDefs(decoder.Reg16HL, b))
		b.NewStore(cg.readMemory(b, hl), cg.findReg8GlobalDef(instr.Reg8Dest))
	})
}

func (cg *Codegen) ld_hl_r8(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		hl := cg.readReg16(b, cg.findReg16GlobalDefs(decoder.Reg16HL, b))
		src := b.NewLoad(types.I8, cg.findReg8GlobalDef(instr.Reg8Src))
		cg.updateMemory(b, hl, src)
	})
}

func (cg *Codegen) ld_hl_n(instr *decoder.Instruction) *ir.Func {
	return cg.buildParamFunc(instr, ir.NewParam("n", types.I8), func(b *ir.Block, p *ir.Param) {
		hl := cg.readReg16(b, cg.findReg16GlobalDefs(decoder.Reg16HL, b))
		cg.updateMemory(b, hl, p)
	})
}

func (cg *Codegen) ld_a_bc(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		bc := cg.readReg16(b, cg.findReg16GlobalDefs(decoder.Reg16BC, b))
		b.NewStore(cg.readMemory(b, bc), cg.aReg)
	})
}

func (cg *Codegen) ld_a_de(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		de := cg.readReg16(b, cg.findReg16GlobalDefs(decoder.Reg16DE, b))
		b.NewStore(cg.readMemory(b, de), cg.aReg)
	})
}

func (cg *Codegen) ld_bc_a(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		bc := cg.readReg16(b, cg.findReg16GlobalDefs(decoder.Reg16BC, b))
		a := b.NewLoad(types.I8, cg.aReg)
		cg.updateMemory(b, bc, a)
	})
}

func (cg *Codegen) ld_de_a(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		de := cg.readReg16(b, cg.findReg16GlobalDefs(decoder.Reg16DE, b))
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
		r16 := cg.findReg16GlobalDefs(decoder.Reg16HL, b)
		hl := cg.readReg16(b, r16)
		b.NewStore(cg.readMemory(b, hl), cg.aReg)
		hlDec := b.NewSub(hl, constant.NewInt(types.I16, 1))
		cg.updateReg16(b, r16, hlDec)
	})
}

func (cg *Codegen) ld_hl_dec_a(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		r16 := cg.findReg16GlobalDefs(decoder.Reg16HL, b)
		hl := cg.readReg16(b, r16)
		a := b.NewLoad(types.I8, cg.aReg)
		cg.updateMemory(b, hl, a)
		hlDec := b.NewSub(hl, constant.NewInt(types.I16, 1))
		cg.updateReg16(b, r16, hlDec)
	})
}

func (cg *Codegen) ld_a_hl_inc(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		r16 := cg.findReg16GlobalDefs(decoder.Reg16HL, b)
		hl := cg.readReg16(b, r16)
		b.NewStore(cg.readMemory(b, hl), cg.aReg)
		hlInc := b.NewAdd(hl, constant.NewInt(types.I16, 1))
		cg.updateReg16(b, r16, hlInc)
	})
}

func (cg *Codegen) ld_hl_inc_a(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		r16 := cg.findReg16GlobalDefs(decoder.Reg16HL, b)
		hl := cg.readReg16(b, r16)
		a := b.NewLoad(types.I8, cg.aReg)
		cg.updateMemory(b, hl, a)
		hlInc := b.NewAdd(hl, constant.NewInt(types.I16, 1))
		cg.updateReg16(b, r16, hlInc)
	})
}

func (cg *Codegen) ld_r16_nn(instr *decoder.Instruction) *ir.Func {
	return cg.buildParamFunc(instr, ir.NewParam("nn", types.I16), func(b *ir.Block, p *ir.Param) {
		r16 := cg.findReg16GlobalDefs(instr.Reg16, b)
		cg.updateReg16(b, r16, p)
	})
}

func (cg *Codegen) ld_nn_sp(instr *decoder.Instruction) *ir.Func {
	return cg.buildParamFunc(instr, ir.NewParam("nn", types.I16), func(b *ir.Block, p *ir.Param) {
		sp := cg.readReg16(b, cg.findReg16GlobalDefs(decoder.Reg16SP, b))
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
		sp := cg.findReg16GlobalDefs(decoder.Reg16SP, b)
		hl := cg.readReg16(b, cg.findReg16GlobalDefs(decoder.Reg16HL, b))
		cg.updateReg16(b, sp, hl)
	})
}

func (cg *Codegen) ld_hl_sp_e(instr *decoder.Instruction) *ir.Func {
	return cg.buildParamFunc(instr, ir.NewParam("e", types.I8), func(b *ir.Block, p *ir.Param) {
		sp := cg.readReg16(b, cg.findReg16GlobalDefs(decoder.Reg16SP, b))
		hl := cg.findReg16GlobalDefs(decoder.Reg16HL, b)

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
		r16 := cg.findReg16GlobalDefs(instr.Reg16, b)
		sp := cg.findReg16GlobalDefs(decoder.Reg16SP, b)
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
		r16 := cg.findReg16GlobalDefs(instr.Reg16, b)
		sp := cg.findReg16GlobalDefs(decoder.Reg16SP, b)
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

func (cg *Codegen) jp_nn(instr *decoder.Instruction, irBlock *ir.Block) error {
	cg.increaseCycles(instr, irBlock)
	toBlock, ok := cg.irBlocks[instr.Imm16Bit]
	if !ok {
		return fmt.Errorf("cannot find jp nn destination block")
	}

	irBlock.NewBr(toBlock)
	return nil
}
