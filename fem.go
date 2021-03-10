// +build ignore

package main

import (
	"fmt"

	"github.com/Konstantin8105/sm"
)

func main() {
	sm.MaxIteration = -1

	truss()
	// d2()
}
func truss() {
	Q, err := sm.Sexpr(nil, "matrix(1, x, 1,2);")
	if err != nil {
		panic(err)
	}
	fmt.Println("Q = ", Q)

	C, err := sm.Sexpr(nil, "matrix(1,0,1,l,2,2)")
	if err != nil {
		panic(err)
	}
	fmt.Println("C = ", C)

	inverseC, err := sm.Sexpr(nil, "inverse("+C+")")
	if err != nil {
		panic(err)
	}
	fmt.Println("C^-1 = ", inverseC)

	bb := Q + "*" + inverseC
	fmt.Println("bb = ", bb)

	B, err := sm.Sexpr(nil, bb)
	if err != nil {
		panic(err)
	}
	fmt.Println("B = ", B)
	//
	// 	E, err := sm.Sexpr(nil, "integral( transpose("+B+")*"+B+",x,0,l); constant(l)")
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	fmt.Println("E = ", E)
}

func d2() {
	Q, err := sm.Sexpr(nil,
		`matrix(
 			1, x, 0, 0, 0, 0,
 			0, 0, 1, x, x*x, pow(x,3),
 			0, 0, 0, 1, 2*x, 3*x*x,
 			3,6);
 		`,
	)
	if err != nil {
		panic(err)
	}
	fmt.Println("Q = ", Q)

	C, err := sm.Sexpr(nil, `
 	matrix(
 		1,0,0,0,0,0,
 		0,0,1,0,0,0,
 		0,0,0,1,0,0,
 		1,l,0,0,0,0,
 		0,0,1,l,l*l,l*l*l,
 		0,0,0,1,2*l,3*l*l,
 		6,6);
 	`)
	if err != nil {
		panic(err)
	}
	fmt.Println("C = ", C)

	inverseC, err := sm.Sexpr(nil, "inverse("+C+")")
	if err != nil {
		panic(err)
	}
	fmt.Println("C^-1 = ", inverseC)

	f, err := sm.Sexpr(nil, Q+` *
	 	matrix(a1,a2,a3,a4,a5,a6, 6,1)
	 `)
	if err != nil {
		panic(err)
	}
	fmt.Println("f = ", f)

	bb := Q + "*" + inverseC
	fmt.Println("bb = ", bb)

	B, err := sm.Sexpr(nil, bb)
	if err != nil {
		panic(err)
	}
	fmt.Println("B = ", B)
}
