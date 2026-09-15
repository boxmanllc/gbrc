package codegen

import (
	"fmt"
	"math"
	"os"

	"github.com/0xmukesh/boxman/internal/analyzer"
	"github.com/0xmukesh/boxman/internal/decoder"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/enum"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

type Codegen struct {
	module *ir.Module
	main   *ir.Func

	instrFuncs map[string]*ir.Func
	irBlocks   map[uint16]*ir.Block

	retDispatchBlock *ir.Block

	ram                                      *ir.Global
	cycles                                   *ir.Global
	aReg, bReg, cReg, dReg, eReg, hReg, lReg *ir.Global
	zFlag, nFlag, hFlag, cFlag               *ir.Global
	pc, sp                                   *ir.Global
	ime                                      *ir.Global

	gBudget *ir.Global

	readRam   *ir.Func
	writeRam  *ir.Func
	readMem   *ir.Func
	interpRun *ir.Func
	intSvc    *ir.Func

	debug     bool
	debugFunc *ir.Func
}

type function struct {
	irFunc *ir.Func
	args   []value.Value
}

func New(blocks []*analyzer.Block, romBytes []byte, debug bool) (*Codegen, error) {
	cg := &Codegen{
		instrFuncs: make(map[string]*ir.Func),
		irBlocks:   make(map[uint16]*ir.Block),
	}

	cg.module = ir.NewModule()
	cg.main = cg.module.NewFunc("rom_main", types.I32)
	cg.debug = debug

	cg.emitGlobals(romBytes)
	if cg.debug {
		cg.setupDebugFunc()
	}

	bootEntry := cg.main.NewBlock("boot_entry")

	for _, block := range blocks {
		if err := cg.emitBlock(block); err != nil {
			return nil, err
		}
	}

	if err := cg.setupReturnDispatcher(blocks); err != nil {
		return nil, err
	}

	for _, block := range blocks {
		if err := cg.joinBlocks(block); err != nil {
			return nil, err
		}
	}

	if len(blocks) > 0 {

		bootEntry.NewBr(cg.retDispatchBlock)
	} else {
		bootEntry.NewRet(constant.NewInt(types.I32, 0))
	}

	return cg, nil
}

func (cg *Codegen) WriteTo(filepath string) error {
	return os.WriteFile(filepath, []byte(cg.module.String()), 0644)
}

func (cg *Codegen) emitGlobals(romBytes []byte) {
	image := make([]byte, 0x10000)
	copy(image, romBytes)
	cg.ram = cg.module.NewGlobalDef("ram", constant.NewCharArray(image))
	cg.cycles = cg.module.NewGlobalDef("cycles", constant.NewInt(types.I32, 0))

	cg.aReg = cg.module.NewGlobalDef("a_reg", constant.NewInt(types.I8, 0))
	cg.bReg = cg.module.NewGlobalDef("b_reg", constant.NewInt(types.I8, 0))
	cg.cReg = cg.module.NewGlobalDef("c_reg", constant.NewInt(types.I8, 0))
	cg.dReg = cg.module.NewGlobalDef("d_reg", constant.NewInt(types.I8, 0))
	cg.eReg = cg.module.NewGlobalDef("e_reg", constant.NewInt(types.I8, 0))
	cg.hReg = cg.module.NewGlobalDef("h_reg", constant.NewInt(types.I8, 0))
	cg.lReg = cg.module.NewGlobalDef("l_reg", constant.NewInt(types.I8, 0))

	cg.zFlag = cg.module.NewGlobalDef("z_flag", constant.NewBool(false))
	cg.nFlag = cg.module.NewGlobalDef("n_flag", constant.NewBool(false))
	cg.hFlag = cg.module.NewGlobalDef("h_flag", constant.NewBool(false))
	cg.cFlag = cg.module.NewGlobalDef("c_flag", constant.NewBool(false))

	cg.pc = cg.module.NewGlobalDef("pc", constant.NewInt(types.I16, 0))
	cg.sp = cg.module.NewGlobalDef("sp", constant.NewInt(types.I16, 0))
	cg.ime = cg.module.NewGlobal("IME", types.I8)
	cg.ime.Linkage = enum.LinkageExternal

	cg.gBudget = cg.module.NewGlobalDef("g_budget", constant.NewInt(types.I32, math.MaxUint32))
	cg.interpRun = cg.module.NewFunc("interp_run", types.I16, ir.NewParam("pc", types.I16))
	cg.intSvc = cg.module.NewFunc("interrupt_service", types.I16)

	cg.readRam = cg.module.NewFunc("read_ram", types.I8, ir.NewParam("addr", types.I16))
	cg.writeRam = cg.module.NewFunc("write_ram", types.Void, ir.NewParam("addr", types.I16), ir.NewParam("val", types.I8))

	cg.setupReadMemFunc()
}

func (cg *Codegen) setupReadMemFunc() {
	cg.readMem = cg.module.NewFunc("cg_read_memory", types.I8, ir.NewParam("addr", types.I16))

	entry := cg.readMem.NewBlock("entry")
	inlineBlock := cg.readMem.NewBlock("inline")
	ioBlock := cg.readMem.NewBlock("io")
	exitBlock := cg.readMem.NewBlock("exit")

	addr := cg.readMem.Params[0]

	isBus := entry.NewICmp(enum.IPredUGE, addr, constant.NewInt(types.I16, 0xE000))
	entry.NewCondBr(isBus, ioBlock, inlineBlock)

	inlineVal := inlineBlock.NewLoad(types.I8, cg.getRamPtr(inlineBlock, addr))
	inlineBlock.NewBr(exitBlock)

	ioVal := ioBlock.NewCall(cg.readRam, addr)
	ioBlock.NewBr(exitBlock)

	val := exitBlock.NewPhi(
		ir.NewIncoming(ioVal, ioBlock),
		ir.NewIncoming(inlineVal, inlineBlock),
	)
	exitBlock.NewRet(val)
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

		if analyzer.IsBlockTerminator(instr) {
			continue
		}

		fn, err := cg.emitInstruction(instr)
		if err != nil {
			return err
		}

		irBlock.NewCall(fn.irFunc, fn.args...)

		pcVal := irBlock.NewLoad(types.I16, cg.pc)
		pcInc := irBlock.NewAdd(pcVal, constant.NewInt(types.I16, int64(instr.Length)))
		irBlock.NewStore(pcInc, cg.pc)
	}

	return nil
}

func (cg *Codegen) emitInstruction(instr *decoder.Instruction) (*function, error) {

	irFunc, ok := cg.instrFuncs[instr.Mnemonic]
	if !ok {
		var err error
		switch instr.InstructionType {
		case decoder.NOP:
			irFunc, err = cg.nop(instr)
		case decoder.HALT, decoder.STOP:
			irFunc, err = cg.nop(instr)
		case decoder.LD_R8_R8:
			irFunc, err = cg.ld_r8_r8(instr)
		case decoder.LD_R8_N:
			irFunc, err = cg.ld_r8_n(instr)
		case decoder.LD_R8_HL:
			irFunc, err = cg.ld_r8_hl(instr)
		case decoder.LD_HL_R8:
			irFunc, err = cg.ld_hl_r8(instr)
		case decoder.LD_HL_N:
			irFunc, err = cg.ld_hl_n(instr)
		case decoder.LD_A_BC:
			irFunc, err = cg.ld_a_bc(instr)
		case decoder.LD_A_DE:
			irFunc, err = cg.ld_a_de(instr)
		case decoder.LD_BC_A:
			irFunc, err = cg.ld_bc_a(instr)
		case decoder.LD_DE_A:
			irFunc, err = cg.ld_de_a(instr)
		case decoder.LD_A_NN:
			irFunc, err = cg.ld_a_nn(instr)
		case decoder.LD_NN_A:
			irFunc, err = cg.ld_nn_a(instr)
		case decoder.LDH_A_C:
			irFunc, err = cg.ldh_a_c(instr)
		case decoder.LDH_C_A:
			irFunc, err = cg.ldh_c_a(instr)
		case decoder.LDH_A_N:
			irFunc, err = cg.ldh_a_n(instr)
		case decoder.LDH_N_A:
			irFunc, err = cg.ldh_n_a(instr)
		case decoder.LD_A_HL_DEC:
			irFunc, err = cg.ld_a_hl_dec(instr)
		case decoder.LD_HL_DEC_A:
			irFunc, err = cg.ld_hl_dec_a(instr)
		case decoder.LD_A_HL_INC:
			irFunc, err = cg.ld_a_hl_inc(instr)
		case decoder.LD_HL_INC_A:
			irFunc, err = cg.ld_hl_inc_a(instr)
		case decoder.LD_R16_NN:
			irFunc, err = cg.ld_r16_nn(instr)
		case decoder.LD_NN_SP:
			irFunc, err = cg.ld_nn_sp(instr)
		case decoder.LD_SP_HL:
			irFunc, err = cg.ld_sp_hl(instr)
		case decoder.LD_HL_SP_E:
			irFunc, err = cg.ld_hl_sp_e(instr)
		case decoder.PUSH_R16:
			irFunc, err = cg.push_r16(instr)
		case decoder.POP_R16:
			irFunc, err = cg.pop_r16(instr)
		case decoder.ADD_R8, decoder.ADC_R8,
			decoder.SUB_R8, decoder.SBC_R8, decoder.CP_R8,
			decoder.INC_R8, decoder.DEC_R8,
			decoder.AND_R8, decoder.OR_R8, decoder.XOR_R8:
			irFunc, err = cg.bit8_arithmetic_r8(instr)
		case decoder.ADD_HL, decoder.ADC_HL,
			decoder.SUB_HL, decoder.SBC_HL, decoder.CP_HL,
			decoder.INC_HL, decoder.DEC_HL,
			decoder.AND_HL, decoder.OR_HL, decoder.XOR_HL:
			irFunc, err = cg.bit8_arithmetic_hl(instr)
		case decoder.ADD_N, decoder.ADC_N,
			decoder.SUB_N, decoder.SBC_N, decoder.CP_N,
			decoder.AND_N, decoder.OR_N, decoder.XOR_N:
			irFunc, err = cg.bit8_arithmetic_n(instr)
		case decoder.CCF:
			irFunc, err = cg.ccf(instr)
		case decoder.SCF:
			irFunc, err = cg.scf(instr)
		case decoder.CPL:
			irFunc, err = cg.cpl(instr)
		case decoder.DAA:
			irFunc, err = cg.daa(instr)
		case decoder.DI:
			irFunc, err = cg.di(instr)
		case decoder.EI:
			irFunc, err = cg.ei(instr)
		case decoder.INC_R16:
			irFunc, err = cg.inc_r16(instr)
		case decoder.DEC_R16:
			irFunc, err = cg.dec_r16(instr)
		case decoder.ADD_HL_R16:
			irFunc, err = cg.add_hl_r16(instr)
		case decoder.ADD_SP_E:
			irFunc, err = cg.add_sp_e(instr)
		case decoder.RLCA, decoder.RRCA, decoder.RLA,
			decoder.RRA, decoder.CB_RLC_R8, decoder.CB_RLC_HL,
			decoder.CB_RRC_R8, decoder.CB_RRC_HL, decoder.CB_RL_R8,
			decoder.CB_RL_HL, decoder.CB_RR_R8, decoder.CB_RR_HL,
			decoder.CB_SLA_R8, decoder.CB_SLA_HL,
			decoder.CB_SRA_R8, decoder.CB_SRA_HL,
			decoder.CB_SRL_R8, decoder.CB_SRL_HL,
			decoder.CB_SWAP_R8, decoder.CB_SWAP_HL:
			irFunc, err = cg.bitwise(instr)
		case decoder.CB_BIT_R8, decoder.CB_BIT_HL,
			decoder.CB_RES_R8, decoder.CB_RES_HL,
			decoder.CB_SET_R8, decoder.CB_SET_HL:
			irFunc, err = cg.bit_op(instr)
		default:
			return nil, fmt.Errorf("cannot emit opcode function ir. unknown instruction type: %d", instr.InstructionType)
		}
		if err != nil {
			return nil, err
		}

		cg.instrFuncs[instr.Mnemonic] = irFunc
	}

	args := []value.Value{}
	switch instr.InstructionType {
	case decoder.LD_R8_N, decoder.LD_HL_N,
		decoder.LDH_A_N, decoder.LDH_N_A, decoder.LD_HL_SP_E,
		decoder.ADD_SP_E, decoder.ADD_N, decoder.ADC_N,
		decoder.SUB_N, decoder.SBC_N, decoder.CP_N,
		decoder.AND_N, decoder.OR_N, decoder.XOR_N:
		args = []value.Value{constant.NewInt(types.I8, int64(instr.Imm8Bit))}
	case decoder.CB_BIT_R8, decoder.CB_BIT_HL,
		decoder.CB_RES_R8, decoder.CB_RES_HL,
		decoder.CB_SET_R8, decoder.CB_SET_HL:
		args = []value.Value{constant.NewInt(types.I8, int64(instr.BitOpIndex))}
	case decoder.LD_A_NN, decoder.LD_NN_A,
		decoder.LD_R16_NN, decoder.LD_NN_SP:
		args = []value.Value{constant.NewInt(types.I16, int64(instr.Imm16Bit))}
	}

	return &function{
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
		return cg.jp_nn(lastInstr, irBlock)
	case decoder.JP_CC_NN:
		return cg.jp_cc_nn(lastInstr, irBlock)
	case decoder.JR_E:
		return cg.jr_e(lastInstr, irBlock)
	case decoder.JR_CC_E:
		return cg.jr_cc_e(lastInstr, irBlock)
	case decoder.CALL_NN:
		return cg.call_nn(lastInstr, irBlock)
	case decoder.CALL_CC_NN:
		return cg.call_cc_nn(lastInstr, irBlock)
	case decoder.RET:
		return cg.ret(lastInstr, irBlock)
	case decoder.RETI:
		return cg.reti(lastInstr, irBlock)
	case decoder.RET_CC:
		return cg.ret_cc(lastInstr, irBlock)
	case decoder.JP_HL:
		return cg.jp_hl(lastInstr, irBlock)
	case decoder.RST_N:
		return cg.rst_n(lastInstr, irBlock)
	default:

		if len(block.Successors) == 1 {
			irBlock.NewStore(constant.NewInt(types.I16, int64(block.Successors[0])), cg.pc)
			irBlock.NewBr(cg.retDispatchBlock)
			return nil
		}
		irBlock.NewRet(constant.NewInt(types.I32, 0))
	}

	return nil
}

func (cg *Codegen) setupReturnDispatcher(blocks []*analyzer.Block) error {
	starts := make([]uint16, 0, len(blocks))
	for _, block := range blocks {
		starts = append(starts, block.Start)
	}
	if err := cg.emitBlockStarts(starts); err != nil {
		return err
	}

	dispatch := cg.main.NewBlock("ret_dispatch")
	cg.retDispatchBlock = dispatch

	budgetOut := cg.main.NewBlock("ret_budget_out")
	budgetOut.NewRet(constant.NewInt(types.I32, 0))

	cyc := dispatch.NewLoad(types.I32, cg.cycles)
	bud := dispatch.NewLoad(types.I32, cg.gBudget)
	over := dispatch.NewICmp(enum.IPredUGE, cyc, bud)

	intEntry := cg.main.NewBlock("ret_dispatch_int")
	dispatch.NewCondBr(over, budgetOut, intEntry)

	ime := intEntry.NewLoad(types.I8, cg.ime)
	imeSet := intEntry.NewICmp(enum.IPredNE, ime, constant.NewInt(types.I8, 0))
	intService := cg.main.NewBlock("ret_dispatch_int_service")
	lookup := cg.main.NewBlock("ret_dispatch_lookup")
	intEntry.NewCondBr(imeSet, intService, lookup)

	vec := intService.NewCall(cg.intSvc)
	isSvc := intService.NewICmp(enum.IPredNE, vec, constant.NewInt(types.I16, 0xFFFF))
	loadVector := cg.main.NewBlock("ret_dispatch_load_vector")
	intService.NewCondBr(isSvc, loadVector, lookup)
	loadVector.NewStore(vec, cg.pc)
	loadVector.NewBr(lookup)

	cur := lookup
	for i, block := range blocks {
		target, ok := cg.irBlocks[block.Start]
		if !ok {
			return fmt.Errorf("can't find equivalent ir block for 0x%04X block", block.Start)
		}

		var next *ir.Block
		if i == len(blocks)-1 {
			next = cg.main.NewBlock("ret_dispatch_end")
		} else {
			next = cg.main.NewBlock(fmt.Sprintf("ret_dispatch_next_%04X", block.Start))
		}

		pcVal := cur.NewLoad(types.I16, cg.pc)
		cmp := cur.NewICmp(enum.IPredEQ, pcVal, constant.NewInt(types.I16, int64(block.Start)))
		cur.NewCondBr(cmp, target, next)
		cur = next
	}

	pcVal := cur.NewLoad(types.I16, cg.pc)
	newPC := cur.NewCall(cg.interpRun, pcVal)
	cur.NewStore(newPC, cg.pc)
	cur.NewBr(dispatch)
	return nil
}

func (cg *Codegen) emitBlockStarts(starts []uint16) error {
	elems := make([]constant.Constant, 0, len(starts)+1)
	for _, start := range starts {
		elems = append(elems, constant.NewInt(types.I16, int64(start)))
	}

	elems = append(elems, constant.NewInt(types.I16, 0xFFFF))
	typ := types.NewArray(uint64(len(starts)+1), types.I16)
	cg.module.NewGlobalDef("block_starts", constant.NewArray(typ, elems...))
	return nil
}
