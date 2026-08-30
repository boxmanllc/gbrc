package codegen

import (
	"fmt"
	"regexp"

	"github.com/0xmukesh/boxman/internal/decoder"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/enum"
	"github.com/llir/llvm/ir/types"
)

var (
	mnemonicPat = regexp.MustCompile(`[ ,()]`)
)

func (cg *Codegen) mnemonicToFuncName(mnemonic string) string {
	return mnemonicPat.ReplaceAllString(mnemonic, "_")
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

func (cg *Codegen) setupFlagDumpFunc() {
	printfFunc := cg.module.NewFunc("printf", types.I32, ir.NewParam("", types.NewPointer(types.I8)))
	printfFunc.Sig.Variadic = true

	fmtStr := constant.NewCharArrayFromString("A=%02X B=%02X C=%02X D=%02X E=%02X H=%02X L=%02X F=%c%c%c%c cycles=%d\n\x00")
	fmtGlobal := cg.module.NewGlobalDef("fmt", fmtStr)
	fmtGlobal.Linkage = enum.LinkagePrivate
	fmtGlobal.Immutable = true

	dumpFn := cg.module.NewFunc("dump", types.Void)
	cg.dumpFunc = dumpFn
	entry := dumpFn.NewBlock("entry")

	aReg := entry.NewLoad(types.I8, cg.aReg)
	bReg := entry.NewLoad(types.I8, cg.bReg)
	cReg := entry.NewLoad(types.I8, cg.cReg)
	dReg := entry.NewLoad(types.I8, cg.dReg)
	eReg := entry.NewLoad(types.I8, cg.eReg)
	hReg := entry.NewLoad(types.I8, cg.hReg)
	lReg := entry.NewLoad(types.I8, cg.lReg)
	zFlag := entry.NewLoad(types.I1, cg.zFlag)
	nFlag := entry.NewLoad(types.I1, cg.nFlag)
	hFlag := entry.NewLoad(types.I1, cg.hFlag)
	cFlag := entry.NewLoad(types.I1, cg.cFlag)
	cycles := entry.NewLoad(types.I32, cg.cycles)

	zChar := entry.NewSelect(zFlag, constant.NewInt(types.I8, 'Z'), constant.NewInt(types.I8, '-'))
	nChar := entry.NewSelect(nFlag, constant.NewInt(types.I8, 'N'), constant.NewInt(types.I8, '-'))
	hChar := entry.NewSelect(hFlag, constant.NewInt(types.I8, 'H'), constant.NewInt(types.I8, '-'))
	cChar := entry.NewSelect(cFlag, constant.NewInt(types.I8, 'C'), constant.NewInt(types.I8, '-'))

	zChar32 := entry.NewZExt(zChar, types.I32)
	nChar32 := entry.NewZExt(nChar, types.I32)
	hChar32 := entry.NewZExt(hChar, types.I32)
	cChar32 := entry.NewZExt(cChar, types.I32)

	a32 := entry.NewZExt(aReg, types.I32)
	b32 := entry.NewZExt(bReg, types.I32)
	c32 := entry.NewZExt(cReg, types.I32)
	d32 := entry.NewZExt(dReg, types.I32)
	e32 := entry.NewZExt(eReg, types.I32)
	h32 := entry.NewZExt(hReg, types.I32)
	l32 := entry.NewZExt(lReg, types.I32)

	zero := constant.NewInt(types.I32, 0)
	fmtPtr := entry.NewGetElementPtr(fmtStr.Type(), fmtGlobal, zero, zero)

	entry.NewCall(printfFunc, fmtPtr,
		a32, b32, c32, d32, e32, h32, l32,
		zChar32, nChar32, hChar32, cChar32, cycles,
	)
	entry.NewRet(nil)
}
