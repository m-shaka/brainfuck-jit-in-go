package main

import (
	"bufio"
	"fmt"
	"github.com/m-shaka/brainfuck-jit/internal/stack"
	"os"
	"strings"
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
	default:
		panic("invalid op")
	}
}

type program struct {
	instructions []rune
}

func parse(filename string) program {
	var insts []rune
	tokens := "><+-.,[]"
	fp, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer fp.Close()
	scanner := bufio.NewScanner(fp)
	for scanner.Scan() {
		for _, c := range scanner.Text() {
			if strings.Contains(tokens, string(c)) {
				insts = append(insts, c)
			}
		}
	}
	return program{insts}
}

func translate(p program) []bfOp {
	pc := 0
	programSize := len(p.instructions)
	loopStack := stack.NewStack()
	var ops []bfOp
	for pc < programSize {
		instruction := p.instructions[pc]
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
			ops[offset].argument = len(ops)
			ops = append(ops, bfOp{jumpIfDataNotZero, offset})
			pc++
			break
		default:
			start := pc
			pc++
			for pc < programSize && p.instructions[pc] == instruction {
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

func interpret(p program) {
	memory := make([]uint8, memorySize)
	pc := 0
	dataptr := 0
	reader := bufio.NewReader(os.Stdin)
	ops := translate(p)
	for pc < len(ops) {
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
		pc++
	}
}

func main() {
	program := parse(os.Args[1])
	interpret(program)
}
