package codegen

import (
	"fmt"
	"os"

	"github.com/0xmukesh/boxman/internal/analyzer"
	"github.com/0xmukesh/boxman/internal/decoder"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
)

type Codegen struct {
	module     *ir.Module
	main       *ir.Func
	entrypoint *ir.Block
	blocks     []*analyzer.Block
	opcodes    map[decoder.InstructionType]*ir.Func

	aReg, bReg, cReg, dReg, eReg, hReg, lReg *ir.Global
	zFlag, nFlag, hFlag, cFlag               *ir.Global
	pc, sp                                   *ir.Global
	cycles                                   *ir.Global
}

func NewCodegen(blocks []*analyzer.Block) (*Codegen, error) {
	cg := &Codegen{
		opcodes: make(map[decoder.InstructionType]*ir.Func),
	}
	cg.blocks = blocks
	cg.module = ir.NewModule()
	cg.main = cg.module.NewFunc("main", types.I32)
	cg.entrypoint = cg.main.NewBlock("entry")
	cg.entrypoint.NewRet(constant.NewInt(types.I32, 0))

	cg.setupGlobalDefs()
	if err := cg.setupRomEntry(); err != nil {
		return nil, err
	}

	return cg, nil
}

func (cg *Codegen) WriteTo(filepath string) {
	os.WriteFile(filepath, []byte(cg.module.String()), 0644)
}

func (cg *Codegen) setupGlobalDefs() {
	cg.aReg = cg.module.NewGlobalDef("a_reg", constant.NewInt(types.I8, 0))
	cg.bReg = cg.module.NewGlobalDef("b_reg", constant.NewInt(types.I8, 0))
	cg.cReg = cg.module.NewGlobalDef("c_reg", constant.NewInt(types.I8, 0))
	cg.dReg = cg.module.NewGlobalDef("d_reg", constant.NewInt(types.I8, 0))
	cg.eReg = cg.module.NewGlobalDef("e_reg", constant.NewInt(types.I8, 0))
	cg.hReg = cg.module.NewGlobalDef("h_reg", constant.NewInt(types.I8, 0))
	cg.lReg = cg.module.NewGlobalDef("l_reg", constant.NewInt(types.I8, 0))
	cg.zFlag = cg.module.NewGlobalDef("z_flag", constant.NewBool(false)) // zero flag
	cg.nFlag = cg.module.NewGlobalDef("n_flag", constant.NewBool(false)) // subtraction flag
	cg.hFlag = cg.module.NewGlobalDef("h_flag", constant.NewBool(false)) // half carry flag
	cg.cFlag = cg.module.NewGlobalDef("c_flag", constant.NewBool(false)) // carry flag
	cg.pc = cg.module.NewGlobalDef("pc", constant.NewInt(types.I16, 0))
	cg.sp = cg.module.NewGlobalDef("sp", constant.NewInt(types.I16, 0))
	cg.cycles = cg.module.NewGlobalDef("cycles", constant.NewInt(types.I32, 0))
}

func (cg *Codegen) setupRomEntry() error {
	var romEntryBlock *analyzer.Block // block which starts from $100
	for _, block := range cg.blocks {
		if block.Start == 0x100 {
			romEntryBlock = block
			break
		}
	}

	if romEntryBlock == nil {
		return fmt.Errorf("failed to find rom entry block")
	}

	cg.processBlock(romEntryBlock, cg.entrypoint)
	return nil
}

func (cg *Codegen) processBlock(block *analyzer.Block, irBlock *ir.Block) {
	for _, instr := range block.Instructions {
		fn, ok := cg.opcodes[instr.InstructionType]
		if !ok {
			fn = cg.setupInstruction(instr)
		}

		if fn == nil {
			fmt.Printf("not processing %d instr type", instr.InstructionType)
			continue
		}

		irBlock.NewCall(fn)
	}
}

func (cg *Codegen) setupInstruction(instr *decoder.Instruction) *ir.Func {
	var fn *ir.Func

	switch instr.InstructionType {
	case decoder.NOP:
		fn = cg.nop()
	}

	cg.opcodes[instr.InstructionType] = fn
	return fn
}
