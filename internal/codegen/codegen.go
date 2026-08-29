package codegen

import (
	"os"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
)

type Codegen struct {
	module *ir.Module
}

func NewCodegen() *Codegen {
	cg := &Codegen{}
	cg.module = ir.NewModule()
	cg.setupRegisters()

	return cg
}

func (cg *Codegen) setupRegisters() {
	cg.module.NewGlobalDef("gb_a", constant.NewInt(types.I8, 0))
	cg.module.NewGlobalDef("gb_b", constant.NewInt(types.I8, 0))
	cg.module.NewGlobalDef("gb_c", constant.NewInt(types.I8, 0))
	cg.module.NewGlobalDef("gb_d", constant.NewInt(types.I8, 0))
	cg.module.NewGlobalDef("gb_e", constant.NewInt(types.I8, 0))
	cg.module.NewGlobalDef("gb_h", constant.NewInt(types.I8, 0))
	cg.module.NewGlobalDef("gb_l", constant.NewInt(types.I8, 0))
	cg.module.NewGlobalDef("gb_pc", constant.NewInt(types.I16, 0))
	cg.module.NewGlobalDef("gb_sp", constant.NewInt(types.I16, 0))
}

func (cg *Codegen) WriteTo(filepath string) {
	os.WriteFile(filepath, []byte(cg.module.String()), 0644)
}
