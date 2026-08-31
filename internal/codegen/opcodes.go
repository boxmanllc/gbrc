package codegen

import (
	"fmt"

	"github.com/0xmukesh/boxman/internal/decoder"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/types"
)

func (cg *Codegen) nop(instr *decoder.Instruction) *ir.Func {
	fn := cg.module.NewFunc(cg.mnemonicToFuncName(instr.Mnemonic), types.Void)
	entry := fn.NewBlock("entry")
	entry.NewRet(nil)

	cg.increaseCycles(instr, entry)
	return fn
}

func (cg *Codegen) ld_r8_r8(instr *decoder.Instruction) *ir.Func {
	fn := cg.module.NewFunc(cg.mnemonicToFuncName(instr.Mnemonic), types.Void)
	entry := fn.NewBlock("entry")
	entry.NewRet(nil)

	destReg := cg.findReg8GlobalDef(instr.Reg8Dest)
	srcReg := cg.findReg8GlobalDef(instr.Reg8Src)

	srcValue := entry.NewLoad(types.I8, srcReg)
	entry.NewStore(srcValue, destReg)

	cg.increaseCycles(instr, entry)
	return fn
}

func (cg *Codegen) ld_r8_n(instr *decoder.Instruction) *ir.Func {
	nParam := ir.NewParam("n", types.I8)
	fn := cg.module.NewFunc(cg.mnemonicToFuncName(instr.Mnemonic), types.Void, nParam)
	entry := fn.NewBlock("entry")
	entry.NewRet(nil)

	destReg := cg.findReg8GlobalDef(instr.Reg8Dest)
	entry.NewStore(nParam, destReg)

	cg.increaseCycles(instr, entry)
	return fn
}

func (cg *Codegen) ld_r8_hl(instr *decoder.Instruction) *ir.Func {
	fn := cg.module.NewFunc(cg.mnemonicToFuncName(instr.Mnemonic), types.Void)
	entry := fn.NewBlock("entry")
	entry.NewRet(nil)

	destReg := cg.findReg8GlobalDef(instr.Reg8Dest)
	hl := cg.loadHL(entry)
	srcValue := cg.loadMemory(entry, hl)
	entry.NewStore(srcValue, destReg)

	cg.increaseCycles(instr, entry)
	return fn
}

func (cg *Codegen) ld_hl_r8(instr *decoder.Instruction) *ir.Func {
	fn := cg.module.NewFunc(cg.mnemonicToFuncName(instr.Mnemonic), types.Void)
	entry := fn.NewBlock("entry")
	entry.NewRet(nil)

	srcReg := cg.findReg8GlobalDef(instr.Reg8Src)
	hl := cg.loadHL(entry)
	srcValue := entry.NewLoad(types.I8, srcReg)
	cg.storeMemory(entry, hl, srcValue)

	cg.increaseCycles(instr, entry)
	return fn
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
