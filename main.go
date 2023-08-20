//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"os"

	"github.com/Konstantin8105/sm"
)

func main() {
	if len(os.Args) == 0 {
		fmt.Fprintf(os.Stderr, "add expression")
		return
	}

	expr := os.Args[1]
	fmt.Fprintf(os.Stdout, "expr = %s\n", expr)
	var err error

	for iter := 0; iter < 2; iter++ {
		expr, err = sm.Sexpr(os.Stdout, expr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v", err)
			return
		}
	}
	fmt.Fprintf(os.Stdout, "%s\n", expr)
}
