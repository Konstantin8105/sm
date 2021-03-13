// +build ignore

package main

import (
	"fmt"

	"github.com/Konstantin8105/sm"
)

func main() {
	sm.MaxIteration = -1
	sm.FloatFormat = 5

	// truss()
	beam()
	// d2()
}

func cal(name, str string) string {
	fmt.Printf("\nName    : %s\n", name)
	fmt.Printf("\nFormula : %s\n", str)
	val, err := sm.Sexpr(nil, str)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Value     : %s\n", val)
	return val
}

func beam() {
	var (
		w    = cal("displ", "matrix(u1, v1, O1, u2, v2, O2, 6,1)")
		load = cal("load", "matrix( N1, V1, M1, N2, V2, M2, 6,1)")
		a    = cal("a", "matrix(a1,a2,a3,a4,a5,a6,6,1)")
		v    = cal("перемещения v", "transpose("+a+") * matrix(1,x,x*x,x*x*x, 0,0 , 6,1)")
		u    = cal("перемещения u", "transpose("+a+") * matrix(0,0,0,0,1,x, 6,1)")
		L    = cal("[L]", `matrix(
			1,0,0,0,
			0,-1,0,0,
			1,l,l*l,l*l*l,
			0,-1,-2*l,-3*l*l,
			4,4) `)
		inverseL = cal("[L]^-1", "inverse("+L+")")
		Ψbend    = cal("Ψbend", "matrix(1,x,x*x,x*x*x,1,4)*"+inverseL)
		dFdx     = cal("dFdx", "d("+Ψbend+", x); variable(x)")
		d2Fdx2   = cal("d2Fdx2", "d("+dFdx+", x); variable(x); constant(l);")

		Kbend = cal("Kbend", "EJ*integral( transpose("+d2Fdx2+") * "+d2Fdx2+",x,0,l);variable(x); constant(l);")
		K     = cal("K", Kbend+" * l*l*l/EJ")

		Kg  = cal("KG", "N*integral( transpose("+dFdx+") * "+dFdx+",x,0,l);variable(x); constant(l);")
		Kg2 = cal("Kg2", "30*l/N * "+Kg)

		wbend  = cal("displ", "matrix(v1, O1, v2, O2, 4,1)")
		MVbend = cal("MV load", Kbend+" * "+wbend)

		free = cal("free", "matrix(0,1,0,0,4,1)")
		gfactor = cal("gfactor","( " + Kbend + " * " + free + ")" ) // + " * transpose( " + free + "))")

		// + " * -1/(3.99988*EJ/l)")

		// stress = cal("stress", "transpose("+ddispl+") * "+load)
	)
	_ = w
	// _ = displ
	_ = load
	_ = v
	_ = u
	_ = inverseL
	_ = Ψbend
	_ = Kbend
	_ = K
	_ = Kg
	_ = Kg2
	_ = wbend
	_ = MVbend
	_ = gfactor
	//_ = stress
}

func truss() {
	U, err := sm.Sexpr(nil, "a1 + a2*x")
	if err != nil {
		panic(err)
	}
	fmt.Println("U = ", U)

	V, err := sm.Sexpr(nil, "0") // a3 + a4*x + a5*x*x+a6*x*x*x")
	if err != nil {
		panic(err)
	}
	fmt.Println("V = ", V)

	dUdx, err := sm.Sexpr(nil, "d("+U+",x); variable(x)")
	if err != nil {
		panic(err)
	}
	fmt.Println("dUdx = ", dUdx)

	dVdx, err := sm.Sexpr(nil, "d("+V+",x); variable(x)")
	if err != nil {
		panic(err)
	}
	fmt.Println("dVdx = ", dVdx)

	d2Vdx2, err := sm.Sexpr(nil, "d("+dVdx+",x); variable(x)")
	if err != nil {
		panic(err)
	}
	fmt.Println("d2Vdx2 = ", d2Vdx2)

	E, err := sm.Sexpr(nil,
		"EF/2*integral(pow("+dUdx+",2),x,0,l) + "+
			"EJ/2*integral(pow("+d2Vdx2+",2),x,0,l) -  "+
			"P/2 *integral(pow("+dVdx+",2),x,0,l); variable(x);constant(a1,a2);",
	)
	if err != nil {
		panic(err)
	}
	fmt.Println("E = ", E)

	///////////////////

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

	dBdx, err := sm.Sexpr(nil, "d("+B+", x ); constant(l); variable(x)")
	if err != nil {
		panic(err)
	}
	fmt.Println("dB/dx= ", dBdx)

	//
	// 	E, err := sm.Sexpr(nil, "integral( transpose("+B+")*"+B+",x,0,l); constant(l)")
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	fmt.Println("E = ", E)
}

func d2() {
	U, err := sm.Sexpr(nil, "a1 + a2*x")
	if err != nil {
		panic(err)
	}
	fmt.Println("U = ", U)

	V, err := sm.Sexpr(nil, "a3 + a4*x + a5*x*x+a6*x*x*x")
	if err != nil {
		panic(err)
	}
	fmt.Println("V = ", V)

	dVdx, err := sm.Sexpr(nil, "d("+V+",x)")
	if err != nil {
		panic(err)
	}
	fmt.Println("dVdx = ", dVdx)

	d2Vdx2, err := sm.Sexpr(nil, "d("+dVdx+",x)")
	if err != nil {
		panic(err)
	}
	fmt.Println("d2Vdx2 = ", d2Vdx2)

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
