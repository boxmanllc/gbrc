package codegen

import (
	"fmt"
	"os"

	"github.com/0xmukesh/boxman/internal/analyzer"
	"github.com/0xmukesh/boxman/internal/decoder"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

type Codegen struct {
	module *ir.Module
	main   *ir.Func

	instrFuncs map[string]*ir.Func
	irBlocks   map[uint16]*ir.Block

	ram                                      *ir.Global
	cycles                                   *ir.Global
	aReg, bReg, cReg, dReg, eReg, hReg, lReg *ir.Global
	zFlag, nFlag, hFlag, cFlag               *ir.Global
	pc, sp                                   *ir.Global

	dumpFunc *ir.Func
}

type Function struct {
	irFunc *ir.Func
	args   []value.Value
}

func New(blocks []*analyzer.Block, toFlagDump bool) *Codegen {
	cg := &Codegen{
		instrFuncs: make(map[string]*ir.Func),
		irBlocks:   make(map[uint16]*ir.Block),
	}

	cg.module = ir.NewModule()
	cg.main = cg.module.NewFunc("main", types.I32)

	cg.emitGlobals()
	if toFlagDump {
		cg.setupFlagDumpFunc()
	}

	for _, block := range blocks {
		if block.Start < 0x100 {
			continue
		}

		cg.emitBlock(block, toFlagDump)
	}

	for _, block := range blocks {
		if block.Start < 0x100 {
			continue
		}

		cg.joinBlocks(block)
	}

	return cg
}

func (cg *Codegen) WriteTo(filepath string) {
	os.WriteFile(filepath, []byte(cg.module.String()), 0644)
}

func (cg *Codegen) emitGlobals() {
	cg.ram = cg.module.NewGlobalDef("ram", constant.NewZeroInitializer(types.NewArray(0x10000, types.I8)))
	cg.cycles = cg.module.NewGlobalDef("cycles", constant.NewInt(types.I32, 0))
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
}

func (cg *Codegen) emitBlock(block *analyzer.Block, toFlagDump bool) {
	entry := cg.main.NewBlock(fmt.Sprintf("block_%04X", block.Start))
	cg.emitCalls(block, entry)

	if toFlagDump {
		entry.NewCall(cg.dumpFunc)
	}

	cg.irBlocks[block.Start] = entry
}

func (cg *Codegen) emitCalls(block *analyzer.Block, irBlock *ir.Block) {
	for _, instr := range block.Instructions {
		// code for block terminator instructions are generated inline
		if analyzer.IsBlockTerminator(instr) {
			continue
		}

		fn := cg.emitInstruction(instr)
		if fn == nil {
			fmt.Printf("not processing %d instr type\n", instr.InstructionType)
			continue
		}

		irBlock.NewCall(fn.irFunc, fn.args...)
	}
}

func (cg *Codegen) emitInstruction(instr *decoder.Instruction) *Function {
	// build up the actual ir function
	irFunc, ok := cg.instrFuncs[instr.Mnemonic]
	if !ok {
		switch instr.InstructionType {
		case decoder.NOP:
			irFunc = cg.nop(instr)
		case decoder.LD_R8_R8:
			irFunc = cg.ld_r8_r8(instr)
		case decoder.LD_R8_N:
			irFunc = cg.ld_r8_n(instr)
		case decoder.LD_R8_HL:
			irFunc = cg.ld_r8_hl(instr)
		case decoder.LD_HL_R8:
			irFunc = cg.ld_hl_r8(instr)
		}

		cg.instrFuncs[instr.Mnemonic] = irFunc
	}

	// build up the function arguments
	args := []value.Value{}
	switch instr.InstructionType {
	case decoder.LD_R8_N:
		args = []value.Value{constant.NewInt(types.I8, int64(instr.Imm8Bit))}
	}

	return &Function{
		irFunc: irFunc,
		args:   args,
	}
}

func (cg *Codegen) joinBlocks(block *analyzer.Block) error {
	irBlock, ok := cg.irBlocks[block.Start]
	if !ok {
		return fmt.Errorf("can't find equivalent ir block for 0x%04X block", block.Start)
	}

	lastInstr := block.Instructions[len(block.Instructions)-1]

	switch lastInstr.InstructionType {
	case decoder.JP_NN:
		cg.jp_nn(lastInstr, irBlock)
	default:
		irBlock.NewRet(constant.NewInt(types.I32, 0))
	}

	return nil
}
