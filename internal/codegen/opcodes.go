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
		hl := cg.readReg16(b, cg.hReg, cg.lReg)
		b.NewStore(cg.loadMemory(b, hl), cg.findReg8GlobalDef(instr.Reg8Dest))
	})
}

func (cg *Codegen) ld_hl_r8(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		hl := cg.readReg16(b, cg.hReg, cg.lReg)
		src := b.NewLoad(types.I8, cg.findReg8GlobalDef(instr.Reg8Src))
		cg.storeMemory(b, hl, src)
	})
}

func (cg *Codegen) ld_hl_n(instr *decoder.Instruction) *ir.Func {
	return cg.buildParamFunc(instr, ir.NewParam("n", types.I8), func(b *ir.Block, p *ir.Param) {
		hl := cg.readReg16(b, cg.hReg, cg.lReg)
		cg.storeMemory(b, hl, p)
	})
}

func (cg *Codegen) ld_a_bc(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		bc := cg.readReg16(b, cg.bReg, cg.cReg)
		b.NewStore(cg.loadMemory(b, bc), cg.aReg)
	})
}

func (cg *Codegen) ld_a_de(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		de := cg.readReg16(b, cg.dReg, cg.eReg)
		b.NewStore(cg.loadMemory(b, de), cg.aReg)
	})
}

func (cg *Codegen) ld_bc_a(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		bc := cg.readReg16(b, cg.bReg, cg.cReg)
		a := b.NewLoad(types.I8, cg.aReg)
		cg.storeMemory(b, bc, a)
	})
}

func (cg *Codegen) ld_de_a(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		de := cg.readReg16(b, cg.dReg, cg.eReg)
		a := b.NewLoad(types.I8, cg.aReg)
		cg.storeMemory(b, de, a)
	})
}

func (cg *Codegen) ld_a_nn(instr *decoder.Instruction) *ir.Func {
	return cg.buildParamFunc(instr, ir.NewParam("nn", types.I16), func(b *ir.Block, p *ir.Param) {
		b.NewStore(cg.loadMemory(b, p), cg.aReg)
	})
}

func (cg *Codegen) ld_nn_a(instr *decoder.Instruction) *ir.Func {
	return cg.buildParamFunc(instr, ir.NewParam("nn", types.I16), func(b *ir.Block, p *ir.Param) {
		a := b.NewLoad(types.I8, cg.aReg)
		cg.storeMemory(b, p, a)
	})
}

func (cg *Codegen) ldh_a_c(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		addr := cg.calculateHighPageAddress(b, b.NewLoad(types.I8, cg.cReg))
		b.NewStore(cg.loadMemory(b, addr), cg.aReg)
	})
}

func (cg *Codegen) ldh_c_a(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		addr := cg.calculateHighPageAddress(b, b.NewLoad(types.I8, cg.cReg))
		a := b.NewLoad(types.I8, cg.aReg)
		cg.storeMemory(b, addr, a)
	})
}

func (cg *Codegen) ldh_a_n(instr *decoder.Instruction) *ir.Func {
	return cg.buildParamFunc(instr, ir.NewParam("n", types.I8), func(b *ir.Block, p *ir.Param) {
		addr := cg.calculateHighPageAddress(b, p)
		b.NewStore(cg.loadMemory(b, addr), cg.aReg)
	})
}

func (cg *Codegen) ldh_n_a(instr *decoder.Instruction) *ir.Func {
	return cg.buildParamFunc(instr, ir.NewParam("n", types.I8), func(b *ir.Block, p *ir.Param) {
		addr := cg.calculateHighPageAddress(b, p)
		a := b.NewLoad(types.I8, cg.aReg)
		cg.storeMemory(b, addr, a)
	})
}

func (cg *Codegen) ld_a_hl_dec(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		hl := cg.readReg16(b, cg.hReg, cg.lReg)
		b.NewStore(cg.loadMemory(b, hl), cg.aReg)
		hlDec := b.NewSub(hl, constant.NewInt(types.I16, 1))
		cg.updateReg16(b, hlDec, cg.hReg, cg.lReg)
	})
}

func (cg *Codegen) ld_hl_dec_a(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		hl := cg.readReg16(b, cg.hReg, cg.lReg)
		a := b.NewLoad(types.I8, cg.aReg)
		cg.storeMemory(b, hl, a)
		hlDec := b.NewSub(hl, constant.NewInt(types.I16, 1))
		cg.updateReg16(b, hlDec, cg.hReg, cg.lReg)
	})
}

func (cg *Codegen) ld_a_hl_inc(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		hl := cg.readReg16(b, cg.hReg, cg.lReg)
		b.NewStore(cg.loadMemory(b, hl), cg.aReg)
		hlInc := b.NewAdd(hl, constant.NewInt(types.I16, 1))
		cg.updateReg16(b, hlInc, cg.hReg, cg.lReg)
	})
}

func (cg *Codegen) ld_hl_inc_a(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		hl := cg.readReg16(b, cg.hReg, cg.lReg)
		a := b.NewLoad(types.I8, cg.aReg)
		cg.storeMemory(b, hl, a)
		hlInc := b.NewAdd(hl, constant.NewInt(types.I16, 1))
		cg.updateReg16(b, hlInc, cg.hReg, cg.lReg)
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
