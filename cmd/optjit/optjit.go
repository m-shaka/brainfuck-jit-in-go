package main

import (
	"fmt"
	"github.com/m-shaka/brainfuck-jit/internal/stack"
	"github.com/m-shaka/brainfuck-jit/internal/util"
	"os"
	"syscall"
	"unsafe"
)

const mmapFlags = syscall.MAP_ANONYMOUS

const memorySize = 30000

const mask = 0xFF

type machineCode []byte

func (m *machineCode) emitBytes(bs ...byte) {
	*m = append(*m, bs...)
}
func (m *machineCode) emitU16(i int) {
	m.emitBytes(byte(i&mask), byte((i>>8)&mask))
}

func (m *machineCode) emitU32(i int) {
	m.emitU16(i & 0xFFFF)
	m.emitU16(i >> 16 & 0xFFFF)
}

func (m *machineCode) emitU64(i int) {
	m.emitU32(i & 0xFFFFFFFF)
	m.emitU32(i >> 32 & 0xFFFFFFFF)
}

type bfOpKind int

const (
	invalidOp bfOpKind = iota
	incPtr
	decPtr
	incData
	decData
	readStdin
	writeStdout
	loopSetToZero
	loopMovePtr
	loopMoveData
	jumpIfDataZero
	jumpIfDataNotZero
)

type bfOp struct {
	kind     bfOpKind
	argument int
}

func (op *bfOp) toToken() rune {
	switch op.kind {
	case incPtr:
		return '>'
	case decPtr:
		return '<'
	case incData:
		return '+'
	case decData:
		return '-'
	case readStdin:
		return ','
	case writeStdout:
		return '.'
	case jumpIfDataZero:
		return '['
	case jumpIfDataNotZero:
		return ']'
	case loopSetToZero:
		return 's'
	case loopMoveData:
		return 'm'
	case loopMovePtr:
		return 'p'
	default:
		panic("invalid op")
	}
}

func translate(p util.Program) []bfOp {
	pc := 0
	programSize := len(p.Instructions)
	loopStack := stack.NewStack()
	var ops []bfOp
	for pc < programSize {
		instruction := p.Instructions[pc]
		switch instruction {
		case '[':
			loopStack.Push(len(ops))
			ops = append(ops, bfOp{jumpIfDataZero, 0})
			pc++
			break
		case ']':
			offset, err := loopStack.Pop()
			if err != nil {
				panic(fmt.Sprintf("unmatched closing ']' at pc=%d", pc))
			}
			optimizedOps := optimizeLoop(ops, offset)
			if len(optimizedOps) == 0 {
				ops[offset].argument = len(ops)
				ops = append(ops, bfOp{jumpIfDataNotZero, offset})
			} else {
				ops = append(ops[0:offset], optimizedOps...)
			}
			pc++
			break
		default:
			start := pc
			pc++
			for pc < programSize && p.Instructions[pc] == instruction {
				pc++
			}
			numRepeats := pc - start
			var kind bfOpKind
			switch instruction {
			case '>':
				kind = incPtr
				break
			case '<':
				kind = decPtr
				break
			case '+':
				kind = incData
				break
			case '-':
				kind = decData
				break
			case ',':
				kind = readStdin
				break
			case '.':
				kind = writeStdout
				break
			default:
				panic(fmt.Sprintf("bad char '%c' at pc=%d", instruction, pc))
			}
			ops = append(ops, bfOp{kind, numRepeats})
		}
	}
	return ops
}

func optimizeLoop(ops []bfOp, loopStart int) []bfOp {
	var optimizedOps []bfOp
	loopSize := len(ops) - loopStart
	if loopSize == 2 {
		repeatedOp := ops[loopStart+1]
		switch repeatedOp.kind {
		case incData:
		case decData:
			optimizedOps = append(optimizedOps, bfOp{loopSetToZero, 0})
			break
		case incPtr:
		case decPtr:
			arg := repeatedOp.argument
			if repeatedOp.kind == decPtr {
				arg = -arg
			}
			optimizedOps = append(optimizedOps, bfOp{loopMovePtr, arg})
		}
	} else if loopSize == 5 {
		if ops[loopStart+1].kind == decData &&
			ops[loopSize+3].kind == incData &&
			ops[loopStart+1].argument == 1 &&
			ops[loopStart+3].argument == 1 {
			if ops[loopStart+2].kind == incPtr &&
				ops[loopStart+4].kind == decPtr &&
				ops[loopStart+2].argument == ops[loopStart+4].argument {
				optimizedOps = append(optimizedOps, bfOp{loopMoveData, ops[loopStart+2].argument})
			} else if ops[loopStart+2].kind == decPtr &&
				ops[loopStart+4].kind == incPtr &&
				ops[loopStart+2].argument == ops[loopStart+4].argument {
				optimizedOps = append(optimizedOps, bfOp{loopMoveData, -ops[loopStart+2].argument})
			}
		}
	}
	return optimizedOps
}

func compile(prog util.Program) []byte {
	memory := make([]byte, memorySize)
	p := (uintptr)(unsafe.Pointer(&memory[0]))
	bracketStack := stack.NewStack()
	var code machineCode
	code.emitBytes(0x49, 0xBD)
	code.emitU64(int(p))

	ops := translate(prog)
	for pc := 0; pc < len(ops); pc++ {
		op := ops[pc]
		switch op.kind {
		case incPtr:
			if op.argument < 256 {
				code.emitBytes(0x49, 0x83, 0xc5, byte(op.argument))
			} else {
				code.emitBytes(0x49, 0x81, 0xc5)
				code.emitU32(op.argument)
			}
			break
		case decPtr:
			if op.argument < 256 {
				code.emitBytes(0x49, 0x83, 0xed, byte(op.argument))
			} else {
				code.emitBytes(0x49, 0x81, 0xed)
				code.emitU32(op.argument)
			}
			break
		case incData:
			if op.argument < 256 {
				code.emitBytes(0x41, 0x80, 0x45, 0x00, byte(op.argument))
			} else if op.argument < 65536 {
				code.emitBytes(0x66, 0x41, 0x81, 0x45, 0x00)
				code.emitU16(op.argument)
			}
			break
		case decData:
			if op.argument < 256 {
				code.emitBytes(0x41, 0x80, 0x6d, 0x00, byte(op.argument))
			} else if op.argument < 65536 {
				code.emitBytes(0x66, 0x41, 0x81, 0x6d, 0x00)
				code.emitU16(op.argument)
			}
			break
		case writeStdout:
			code.emitBytes(
				0x48, 0xC7, 0xC0, 0x01, 0x00, 0x00, 0x00,
				0x48, 0xC7, 0xC7, 0x01, 0x00, 0x00, 0x00,
				0x4C, 0x89, 0xEE,
				0x48, 0xC7, 0xC2, 0x01, 0x00, 0x00, 0x00,
				0x0F, 0x05,
			)
			break
		case readStdin:
			code.emitBytes(
				0x48, 0xC7, 0xC0, 0x00, 0x00, 0x00, 0x00,
				0x48, 0xC7, 0xC7, 0x00, 0x00, 0x00, 0x00,
				0x4C, 0x89, 0xEE,
				0x48, 0xC7, 0xC2, 0x01, 0x00, 0x00, 0x00,
				0x0F, 0x05,
			)
			break
		case loopSetToZero:
			code.emitBytes(0x41, 0xC6, 0x45, 0x00, 0x00)
			break
		case loopMovePtr:
			// for memory[dataptr] != 0 {
			// 	dataptr += op.argument
			// }
			break
		case loopMoveData:
			// if memory[dataptr] != 0 {
			// 	memory[dataptr+op.argument] = memory[dataptr]
			// 	memory[dataptr] = 0
			// }
			break
		case jumpIfDataZero:
			code.emitBytes(0x41, 0x80, 0x7d, 0x00, 0x00)
			bracketStack.Push(len(code))
			code.emitBytes(0x0F, 0x84)
			code.emitU32(0) // offset。後で埋め直す
			break
		case jumpIfDataNotZero:
			bracketOffset, err := bracketStack.Pop()
			if err != nil {
				panic("mismatch [")
			}
			// cmpb $0, 0(%r13)
			code.emitBytes(0x41, 0x80, 0x7d, 0x00, 0x00)
			jumpBackFrom := len(code) + 6
			jumpBackTo := bracketOffset + 6
			offsetBack := computeRelativeOffset(jumpBackFrom, jumpBackTo)
			code.emitBytes(0x0F, 0x85)
			code.emitU32(int(offsetBack))

			jumpForwardFrom := jumpBackTo
			jumpForwardTo := len(code)
			offsetForward := computeRelativeOffset(jumpForwardFrom, jumpForwardTo)
			for i := 2; i < 6; i++ {
				code[bracketOffset+i] = byte((offsetForward >> ((i - 2) * 8)) & mask)
			}
			break
		default:
			panic(fmt.Sprintf("bad char '%v' at pc=%d", op.toToken(), pc))
		}
	}
	code.emitBytes(0xC3)
	return code
}

func computeRelativeOffset(from int, to int) uint32 {
	return uint32(to - from)
}

func execute(m machineCode, debug bool) int {
	mmapFunc, err := syscall.Mmap(
		-1,
		0,
		len(m),
		syscall.PROT_READ|syscall.PROT_WRITE|syscall.PROT_EXEC, syscall.MAP_PRIVATE|mmapFlags,
	)
	if err != nil {
		fmt.Printf("mmap err: %v", err)
	}
	for i, b := range m {
		mmapFunc[i] = b
	}
	type execFunc func() int
	unsafeFunc := (uintptr)(unsafe.Pointer(&mmapFunc))
	f := *(*execFunc)(unsafe.Pointer(&unsafeFunc))
	value := f()
	if debug {
		fmt.Println("\nResult :", value)
		fmt.Printf("Hex    : %x\n", value)
		fmt.Printf("Size   : %d bytes\n\n", len(m))
	}
	return value
}

func main() {
	program := util.Parse(os.Args[1])
	code := compile(program)
	execute(code, false)
}
