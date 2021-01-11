package main

import (
	"bufio"
	"fmt"
	"github.com/m-shaka/brainfuck-jit/internal/stack"
	"github.com/m-shaka/brainfuck-jit/internal/util"
	"os"
)

const memorySize = 30000

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

func interpret(p util.Program) {
	memory := make([]uint8, memorySize)
	dataptr := 0
	reader := bufio.NewReader(os.Stdin)
	ops := translate(p)
	for pc := 0; pc < len(ops); pc++ {
		op := ops[pc]
		switch op.kind {
		case incPtr:
			dataptr += op.argument
			break
		case decPtr:
			dataptr -= op.argument
			break
		case incData:
			memory[dataptr] += uint8(op.argument)
			break
		case decData:
			memory[dataptr] -= uint8(op.argument)
			break
		case readStdin:
			for i := 0; i < op.argument; i++ {
				val, _ := reader.ReadByte()
				memory[dataptr] = uint8(val)
			}
			break
		case writeStdout:
			for i := 0; i < op.argument; i++ {
				print(string(memory[dataptr]))
			}
			break
		case loopSetToZero:
			memory[dataptr] = 0
			break
		case loopMovePtr:
			for memory[dataptr] != 0 {
				dataptr += op.argument
			}
			break
		case loopMoveData:
			if memory[dataptr] != 0 {
				memory[dataptr+op.argument] = memory[dataptr]
				memory[dataptr] = 0
			}
			break
		case jumpIfDataZero:
			if memory[dataptr] == 0 {
				pc = op.argument
			}
			break
		case jumpIfDataNotZero:
			if memory[dataptr] != 0 {
				pc = op.argument
			}
			break
		default:
			panic(fmt.Sprintf("bad char '%v' at pc=%d", op.toToken(), pc))
		}
	}
}

func main() {
	program := util.Parse(os.Args[1])
	interpret(program)
}
