package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

var sc = bufio.NewScanner(os.Stdin)

type program struct {
	instructions string
}

func parse() string {
	var res []rune
	tokens := "><+-.,[]"
	for sc.Scan() {
		line := sc.Text()
		for _, c := range line {
			if strings.Contains(tokens, string(c)) {
				res = append(res, c)
			}
		}
	}
	return string(res)
}

func main() {
	fmt.Println(parse())
}
