package codegen

import (
	"fmt"
	"regexp"

	"github.com/0xmukesh/boxman/internal/decoder"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
)

func (cg *Codegen) scaffoldInstrFunc(instr *decoder.Instruction) (*ir.Func, *ir.Block) {
	re := regexp.MustCompile(`[ ,()]`)
	name := re.ReplaceAllString(instr.Mnemonic, "_")
	fn := cg.module.NewFunc(fmt.Sprintf("op_%s", name), types.Void)
	entry := fn.NewBlock("entry")
	entry.NewRet(nil)
	return fn, entry
}

func (cg *Codegen) increaseCycles(instr *decoder.Instruction, irBlock *ir.Block) {
	cycles := irBlock.NewLoad(types.I32, cg.cycles)
	cyclesInc := irBlock.NewAdd(cycles, constant.NewInt(types.I32, int64(instr.BaseMCycles)))
	irBlock.NewStore(cyclesInc, cg.cycles)
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
