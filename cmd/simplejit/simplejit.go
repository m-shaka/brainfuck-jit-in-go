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

const mask = 0xFF // 11111111

func compile(prog util.Program) []byte {
	bracketStack := stack.NewStack()
	memory := make([]byte, 30000)
	p := (uintptr)(unsafe.Pointer(&memory[0]))
	machineCodes := []byte{
		0x49,
		0xBD,
		byte(p & 0xFF),
		byte((p >> 8) & mask),
		byte((p >> 16) & mask),
		byte((p >> 24) & mask),
		byte((p >> 32) & mask),
		byte((p >> 40) & mask),
		byte((p >> 48) & mask),
		byte((p >> 56) & mask),
	}

	for _, inst := range prog.Instructions {
		switch inst {
		case '>':
			// inc %r13
			machineCodes = append(machineCodes, 0x49, 0xFF, 0xC5)
			break
		case '<':
			// dec %r13
			machineCodes = append(machineCodes, 0x49, 0xFF, 0xCD)
			break
		case '+':
			// addb $1, 0(%r13)
			machineCodes = append(machineCodes, 0x41, 0x80, 0x45, 0x00, 0x01)
			break
		case '-':
			// subb $1, 0(%r13)
			machineCodes = append(machineCodes, 0x41, 0x80, 0x6D, 0x00, 0x01)
			break
		case '.':
			// mov $1, %rax
			// mov $1, %rdi
			// mov %r13, %rsi
			// mov $1, %rdx
			// syscall
			machineCodes = append(
				machineCodes,
				0x48, 0xC7, 0xC0, 0x01, 0x00, 0x00, 0x00,
				0x48, 0xC7, 0xC7, 0x01, 0x00, 0x00, 0x00,
				0x4C, 0x89, 0xEE,
				0x48, 0xC7, 0xC2, 0x01, 0x00, 0x00, 0x00,
				0x0F, 0x05,
			)
			break
		case ',':
			machineCodes = append(
				machineCodes,
				0x48, 0xC7, 0xC0, 0x00, 0x00, 0x00, 0x00,
				0x48, 0xC7, 0xC7, 0x00, 0x00, 0x00, 0x00,
				0x4C, 0x89, 0xEE,
				0x48, 0xC7, 0xC2, 0x01, 0x00, 0x00, 0x00,
				0x0F, 0x05,
			)
			break
		case '[':
			// cmpb $0, 0(%r13)
			machineCodes = append(
				machineCodes,
				0x41, 0x80, 0x7d, 0x00, 0x00,
			)
			bracketStack.Push(len(machineCodes))
			machineCodes = append(
				machineCodes,
				0x0F, 0x84,
				0x0, 0x0, 0x0, 0x0, // offset。後で埋め直す
			)
			break
		case ']':
			bracketOffset, err := bracketStack.Pop()
			if err != nil {
				panic("mismatch [")
			}
			// cmpb $0, 0(%r13)
			machineCodes = append(
				machineCodes,
				0x41, 0x80, 0x7d, 0x00, 0x00,
			)
			jumpBackFrom := len(machineCodes) + 6
			jumpBackTo := bracketOffset + 6
			offsetBack := computeRelativeOffset(jumpBackFrom, jumpBackTo)
			machineCodes = append(
				machineCodes,
				0x0F, 0x85,
				byte(offsetBack&mask),
				byte((offsetBack>>8)&mask),
				byte((offsetBack>>16)&mask),
				byte((offsetBack>>24)&mask),
			)

			jumpForwardFrom := jumpBackTo
			jumpForwardTo := len(machineCodes)
			offsetForward := computeRelativeOffset(jumpForwardFrom, jumpForwardTo)
			for i := 2; i < 6; i++ {
				machineCodes[bracketOffset+i] = byte((offsetForward >> ((i - 2) * 8)) & mask)
			}
			break
		default:
		}
	}
	machineCodes = append(machineCodes, 0xC3)
	return machineCodes
}

func execute(m []byte, debug bool) int {
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

func computeRelativeOffset(from int, to int) uint32 {
	return uint32(to - from)
}

func main() {
	program := util.Parse(os.Args[1])
	codes := compile(program)
	execute(codes, false)
}
