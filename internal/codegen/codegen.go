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

	instrFuncs map[string]*ir.Func  // mapping mneomnic to opcode's ir function
	irBlocks   map[uint16]*ir.Block // mapping block start address to ir block element

	ram                                      *ir.Global
	cycles                                   *ir.Global
	aReg, bReg, cReg, dReg, eReg, hReg, lReg *ir.Global
	zFlag, nFlag, hFlag, cFlag               *ir.Global
	pc, sp                                   *ir.Global

	readRam  *ir.Func // helper for read_ram() called in runtime
	writeRam *ir.Func // helper for write_ram() called in runtime

	debug     bool
	debugFunc *ir.Func
}

type Function struct {
	irFunc *ir.Func
	args   []value.Value
}

func New(blocks []*analyzer.Block, debug bool) (*Codegen, error) {
	cg := &Codegen{
		instrFuncs: make(map[string]*ir.Func),
		irBlocks:   make(map[uint16]*ir.Block),
	}

	cg.module = ir.NewModule()
	cg.main = cg.module.NewFunc("main", types.I32)
	cg.debug = debug

	cg.emitGlobals()
	if cg.debug {
		cg.setupDebugFunc()
	}

	// NOTE: we're only considering blocks starting from rom entry atm
	// FIXME: need to consider blocks before rom entry which includes various interrupt handlers
	var blocksToProcess []*analyzer.Block
	for _, block := range blocks {
		if block.Start >= analyzer.ROM_ENTRY {
			blocksToProcess = append(blocksToProcess, block)
		}
	}

	for _, block := range blocksToProcess {
		if err := cg.emitBlock(block); err != nil {
			return nil, err
		}
	}

	for _, block := range blocksToProcess {
		if err := cg.joinBlocks(block); err != nil {
			return nil, err
		}
	}

	return cg, nil
}

func (cg *Codegen) WriteTo(filepath string) error {
	return os.WriteFile(filepath, []byte(cg.module.String()), 0644)
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

	cg.readRam = cg.module.NewFunc("read_ram", types.I8, ir.NewParam("addr", types.I16))
	cg.writeRam = cg.module.NewFunc("write_ram", types.Void, ir.NewParam("addr", types.I16), ir.NewParam("val", types.I8))
}

func (cg *Codegen) emitBlock(block *analyzer.Block) error {
	entry := cg.main.NewBlock(fmt.Sprintf("block_%04X", block.Start))
	if err := cg.emitCalls(block, entry); err != nil {
		return err
	}

	if cg.debug {
		entry.NewCall(cg.debugFunc)
	}

	cg.irBlocks[block.Start] = entry
	return nil
}

func (cg *Codegen) emitCalls(block *analyzer.Block, irBlock *ir.Block) error {
	for _, instr := range block.Instructions {
		// for block terminators, the code is placed inline at the end of the block
		if analyzer.IsBlockTerminator(instr) {
			continue
		}

		fn, err := cg.emitInstruction(instr)
		if err != nil {
			return err
		}

		irBlock.NewCall(fn.irFunc, fn.args...)
	}

	return nil
}

func (cg *Codegen) emitInstruction(instr *decoder.Instruction) (*Function, error) {
	// first, build up the opcode's function
	// if it is already present then reuse it
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
		case decoder.LD_HL_N:
			irFunc = cg.ld_hl_n(instr)
		case decoder.LD_A_BC:
			irFunc = cg.ld_a_bc(instr)
		case decoder.LD_A_DE:
			irFunc = cg.ld_a_de(instr)
		case decoder.LD_BC_A:
			irFunc = cg.ld_bc_a(instr)
		case decoder.LD_DE_A:
			irFunc = cg.ld_de_a(instr)
		case decoder.LD_A_NN:
			irFunc = cg.ld_a_nn(instr)
		case decoder.LD_NN_A:
			irFunc = cg.ld_nn_a(instr)
		case decoder.LDH_A_C:
			irFunc = cg.ldh_a_c(instr)
		case decoder.LDH_C_A:
			irFunc = cg.ldh_c_a(instr)
		case decoder.LDH_A_N:
			irFunc = cg.ldh_a_n(instr)
		case decoder.LDH_N_A:
			irFunc = cg.ldh_n_a(instr)
		case decoder.LD_A_HL_DEC:
			irFunc = cg.ld_a_hl_dec(instr)
		case decoder.LD_HL_DEC_A:
			irFunc = cg.ld_hl_dec_a(instr)
		case decoder.LD_A_HL_INC:
			irFunc = cg.ld_a_hl_inc(instr)
		case decoder.LD_HL_INC_A:
			irFunc = cg.ld_hl_inc_a(instr)
		case decoder.LD_R16_NN:
			irFunc = cg.ld_r16_nn(instr)
		case decoder.LD_NN_SP:
			irFunc = cg.ld_nn_sp(instr)
		case decoder.LD_SP_HL:
			irFunc = cg.ld_sp_hl(instr)
		case decoder.LD_HL_SP_E:
			irFunc = cg.ld_hl_sp_e(instr)
		case decoder.PUSH_R16:
			irFunc = cg.push_r16(instr)
		case decoder.POP_R16:
			irFunc = cg.pop_r16(instr)
		case decoder.ADD_R8, decoder.ADC_R8,
			decoder.SUB_R8, decoder.SBC_R8, decoder.CP_R8,
			decoder.INC_R8, decoder.DEC_R8,
			decoder.AND_R8, decoder.OR_R8, decoder.XOR_R8:
			irFunc = cg.bit8_arithmetic_r8(instr)
		case decoder.ADD_HL, decoder.ADC_HL,
			decoder.SUB_HL, decoder.SBC_HL, decoder.CP_HL,
			decoder.INC_HL, decoder.DEC_HL,
			decoder.AND_HL, decoder.OR_HL, decoder.XOR_HL:
			irFunc = cg.bit8_arithmetic_hl(instr)
		case decoder.ADD_N, decoder.ADC_N,
			decoder.SUB_N, decoder.SBC_N, decoder.CP_N,
			decoder.AND_N, decoder.OR_N, decoder.XOR_N:
			irFunc = cg.bit8_arithmetic_n(instr)
		case decoder.CCF:
			irFunc = cg.ccf(instr)
		case decoder.SCF:
			irFunc = cg.scf(instr)
		case decoder.CPL:
			irFunc = cg.cpl(instr)
		default:
			return nil, fmt.Errorf("cannot emit opcode function ir. unknown instruction type: %d", instr.InstructionType)
		}

		cg.instrFuncs[instr.Mnemonic] = irFunc
	}

	// then, build up the function's arguments
	args := []value.Value{}
	switch instr.InstructionType {
	case decoder.LD_R8_N, decoder.LD_HL_N,
		decoder.LDH_A_N, decoder.LDH_N_A, decoder.LD_HL_SP_E,
		decoder.ADD_N, decoder.ADC_N,
		decoder.SUB_N, decoder.SBC_N,
		decoder.CP_N:
		args = []value.Value{constant.NewInt(types.I8, int64(instr.Imm8Bit))}
	case decoder.LD_A_NN, decoder.LD_NN_A,
		decoder.LD_R16_NN, decoder.LD_NN_SP:
		args = []value.Value{constant.NewInt(types.I16, int64(instr.Imm16Bit))}
	}

	return &Function{
		irFunc: irFunc,
		args:   args,
	}, nil
}

func (cg *Codegen) joinBlocks(block *analyzer.Block) error {
	irBlock, ok := cg.irBlocks[block.Start]
	if !ok {
		return fmt.Errorf("can't find equivalent ir block for 0x%04X block", block.Start)
	}

	lastInstr := block.Instructions[len(block.Instructions)-1]

	switch lastInstr.InstructionType {
	case decoder.JP_NN:
		if err := cg.jp_nn(lastInstr, irBlock); err != nil {
			return err
		}
	default:
		irBlock.NewRet(constant.NewInt(types.I32, 0))
	}

	return nil
}
