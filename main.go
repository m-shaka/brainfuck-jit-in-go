package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const memorySize = 30000

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

func interpret(p program, jumptable []int) {
	memory := make([]uint8, memorySize)
	pc := 0
	dataptr := 0
	reader := bufio.NewReader(os.Stdin)
	for pc < len(p.instructions) {
		instruction := p.instructions[pc]
		switch instruction {
		case '>':
			dataptr++
			break
		case '<':
			dataptr--
			break
		case '+':
			memory[dataptr]++
			break
		case '-':
			memory[dataptr]--
			break
		case ',':
			val, _ := reader.ReadByte()
			memory[dataptr] = uint8(val)
			break
		case '.':
			print(string(memory[dataptr]))
			break
		case '[':
			if memory[dataptr] == 0 {
				pc = jumptable[pc]
			}
			break
		case ']':
			if memory[dataptr] != 0 {
				pc = jumptable[pc]
			}
			break
		default:
			panic(fmt.Sprintf("bad char '%s' at pc=%d", string(instruction), pc))
		}
		pc++
	}
}

func computeJumptable(p program) []int {
	pc := 0
	programSize := len(p.instructions)
	jumptable := make([]int, programSize)
	for pc < programSize {
		instruction := p.instructions[pc]
		if instruction == '[' {
			bracketNesting := 1
			seek := pc
			for bracketNesting > 0 && seek-2 < programSize {
				seek++
				if p.instructions[seek] == ']' {
					bracketNesting--
				} else if p.instructions[seek] == '[' {
					bracketNesting++
				}
			}
			if bracketNesting <= 0 {
				jumptable[pc] = seek
				jumptable[seek] = pc
			} else {
				panic(fmt.Sprintf("unmatched '[' at pc=%d", pc))
			}
		}
		pc++
	}
	return jumptable
}

func main() {
	program := parse(os.Args[1])
	jumptable := computeJumptable(program)
	interpret(program, jumptable)
}
