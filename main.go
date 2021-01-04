package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const memorySize = 30000

var sc = bufio.NewScanner(os.Stdin)

type program struct {
	instructions []rune
}

func parse() program {
	var insts []rune
	tokens := "><+-.,[]"
	for sc.Scan() {
		line := sc.Text()
		for _, c := range line {
			if strings.Contains(tokens, string(c)) {
				insts = append(insts, c)
			}
		}
	}
	return program{insts}
}

func interpret(p program) {
	memory := make([]uint8, memorySize)
	pc := 0
	dataptr := 0
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
			if sc.Scan() {
				memory[dataptr] = sc.Text()[0]
			}
			break
		case '.':
			print(string(memory[dataptr]))
			break
		case '[':
			if memory[dataptr] == 0 {
				bracketNesting := 1
				savedPC := pc
				for bracketNesting > 0 && pc < len(p.instructions) {
					pc++
					if p.instructions[pc] == ']' {
						bracketNesting--
					} else if p.instructions[pc] == '[' {
						bracketNesting++
					}
				}
				if bracketNesting <= 0 {
					break
				} else {
					panic(fmt.Sprintf("unmatched '[' at pc=%d", savedPC))
				}
			}
			break
		case ']':
			if memory[dataptr] != 0 {
				bracketNesting := 1
				savedPC := pc
				for bracketNesting > 0 && pc > 0 {
					pc--
					if p.instructions[pc] == '[' {
						bracketNesting--
					} else if p.instructions[pc] == ']' {
						bracketNesting++
					}
				}
				if bracketNesting <= 0 {
					break
				} else {
					panic(fmt.Sprintf("unmatched ']' at pc=%d", savedPC))
				}
			}
			break
		default:
			panic(fmt.Sprintf("bad char '%s' at pc=%d", string(instruction), pc))
		}
		pc++
	}
}

func main() {
	program := parse()
	interpret(program)
}
