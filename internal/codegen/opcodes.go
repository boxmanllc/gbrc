package codegen

import (
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
)

func (cg *Codegen) nop() *ir.Func {
	fn := cg.module.NewFunc("op_NOP", types.Void)
	entry := fn.NewBlock("entry")
	cycles := entry.NewLoad(types.I32, cg.cycles)
	cyclesInc := entry.NewAdd(cycles, constant.NewInt(types.I32, 1))
	entry.NewStore(cyclesInc, cg.cycles)
	entry.NewRet(nil)

	return fn
}
