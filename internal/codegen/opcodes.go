package codegen

import (
	"fmt"

	"github.com/0xmukesh/boxman/internal/decoder"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/types"
)

func (cg *Codegen) buildVoidFunc(instr *decoder.Instruction, build func(*ir.Block)) *ir.Func {
	fn := cg.module.NewFunc(mnemonicToFuncName(instr.Mnemonic), types.Void)
	entry := fn.NewBlock("entry")
	build(entry)
	cg.increaseCycles(instr, entry)
	entry.NewRet(nil)
	return fn
}

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
	nParam := ir.NewParam("n", types.I8)
	fn := cg.module.NewFunc(mnemonicToFuncName(instr.Mnemonic), types.Void, nParam)
	entry := fn.NewBlock("entry")
	entry.NewStore(nParam, cg.findReg8GlobalDef(instr.Reg8Dest))
	cg.increaseCycles(instr, entry)
	entry.NewRet(nil)
	return fn
}

func (cg *Codegen) ld_r8_hl(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		hl := cg.loadHL(b)
		b.NewStore(cg.loadMemory(b, hl), cg.findReg8GlobalDef(instr.Reg8Dest))
	})
}

func (cg *Codegen) ld_hl_r8(instr *decoder.Instruction) *ir.Func {
	return cg.buildVoidFunc(instr, func(b *ir.Block) {
		hl := cg.loadHL(b)
		src := b.NewLoad(types.I8, cg.findReg8GlobalDef(instr.Reg8Src))
		cg.storeMemory(b, hl, src)
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
