package sm

import (
	"bytes"
	"fmt"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"os"
	"strconv"
	"strings"

	goast "go/ast"
)

var maxIteration int64 = 1000000

// Sexpr - simplification of expression.
func Sexpr(out io.Writer, expr string, variables ...string) (re string, err error) {
	if out == nil {
		out = os.Stdout
	}

	var a goast.Expr
	a, err = parser.ParseExpr(expr)
	if err != nil {
		return
	}

	var iter int64
	var changed bool
	for {
		changed, a = walk(a, variables)
		{
			var buf bytes.Buffer
			printer.Fprint(&buf, token.NewFileSet(), a)
			re = buf.String()
			fmt.Fprintf(out, "%s\n", re)
		}
		if !changed {
			break
		}
		if iter > maxIteration {
			err = fmt.Errorf("maximal iteration limit")
			return
		}
		iter++
	}

	// debug
	// goast.Print(token.NewFileSet(), a)

	return
}

var rules []func(goast.Expr, []string) (bool, goast.Expr)

func init() {
	rules = []func(goast.Expr, []string) (bool, goast.Expr){
		constants,             // 0
		constantsLeft,         // 1
		constantsLeftLeft,     // 2
		openParenLeft,         // 3
		openParenRight,        // 4
		openParen,             // 5
		openParenSingleNumber, // 6
		openParenSingleIdent,  // 7
		sortIdentMul,          // 8
		functionPow,           // 9
	}
}

func view(a goast.Expr) {
	var buf bytes.Buffer
	printer.Fprint(&buf, token.NewFileSet(), a)
	fmt.Println(buf.String())
}

var counter int

func walk(a goast.Expr, variables []string) (c bool, _ goast.Expr) {
	// debug
	// var buf bytes.Buffer
	// printer.Fprint(&buf, token.NewFileSet(), a)
	// fmt.Println(counter, "walk:before: ", buf.String())
	// counter++
	// defer func() {
	// var buf bytes.Buffer
	// printer.Fprint(&buf, token.NewFileSet(), a)
	// counter--
	// fmt.Println(counter, "walk:after : ", buf.String(), c)
	// }()

	// try simplification
	{
		var changed bool
		var r goast.Expr
		for i := 0; i < len(rules); i++ {
			// fmt.Println("try rules = ", i)
			var c bool
			c, r = rules[i](a, variables)
			if c {
				// fmt.Println("rules = ", i)
				a = r
				changed = true
				i = 0
			}
			// view(a)
		}
		if changed {
			return changed, a
		}
	}

	// go deeper
	switch v := a.(type) {
	case *goast.BinaryExpr:
		cX, rX := walk(v.X, variables)
		cY, rY := walk(v.Y, variables)
		changed := cX || cY
		v.X = rX
		v.Y = rY
		return changed, v

	case *goast.ParenExpr:
		return walk(v.X, variables)

	case *goast.BasicLit:
		if v.Kind == token.INT {
			return true, createFloat(v.Value)
		}

	case *goast.Ident: // ignore

	case *goast.CallExpr:
		var call goast.CallExpr
		call.Fun = v.Fun
		var changed bool
		for i := range v.Args {
			fmt.Println(i)
			c, e := walk(v.Args[i], variables)
			if c {
				changed = true
				call.Args = append(call.Args, e)
				continue
			}
			call.Args = append(call.Args, v.Args[i])
		}
		if changed {
			return true, &call
		}

	default:
		panic(fmt.Errorf("Add implementation for type %T", a))
	}

	// all is not changed
	return false, a
}

func functionPow(a goast.Expr, variables []string) (changed bool, r goast.Expr) {
	call, ok := a.(*goast.CallExpr)
	if !ok {
		return false, nil
	}
	id, ok := call.Fun.(*goast.Ident)
	if !ok {
		return false, nil
	}
	if id.Name != "pow" {
		return false, nil
	}
	fmt.Println("2")
	if len(call.Args) != 2 {
		panic("function pow have not 2 arguments")
	}

	fmt.Println("22")
	e, ok := call.Args[1].(*goast.BasicLit)
	if !ok {
		return false, nil
	}

	fmt.Println("3")
	exponent, err := strconv.ParseFloat(e.Value, 64)
	if err != nil {
		panic(err)
	}

	fmt.Println("4", exponent, exponent == float64(int64(exponent)))
	if exponent != float64(int64(exponent)) {
		return false, nil
	}

	exn := int64(exponent)

	switch exn {
	case 0:
		// from:
		// pow(..., 0)
		// to:
		// 1
		return true, createFloat("1")

	case 1:
		// from:
		// pow(..., 1)
		// to:
		// (...)
		return true, &goast.ParenExpr{X: call.Args[0]}

	case 2:
		// from:
		// pow(..., 2)
		// to:
		// (...) * (...)
		x1 := call.Args[0]
		x2 := call.Args[0]
		g := &goast.BinaryExpr{
			X:  &goast.ParenExpr{X: x1},
			Op: token.MUL,
			Y:  &goast.ParenExpr{X: x2},
		}
		return true, g
	}

	if exn > 0 {
		// from:
		// pow(..., 33)
		// to:
		// (...) * pow(..., 32)
		x1 := call.Args[0]
		x2 := call.Args[0]
		return true, &goast.BinaryExpr{
			X:  &goast.ParenExpr{X: x1},
			Op: token.MUL,
			Y: &goast.CallExpr{
				Fun: goast.NewIdent("pow"),
				Args: []goast.Expr{
					x2,
					createFloat(fmt.Sprintf("%d", exn-1)),
				},
			},
		}
	}

	// from:
	// pow(..., -33)
	// to:
	// pow(..., -32) / (...)
	x1 := call.Args[0]
	x2 := call.Args[0]
	return true, &goast.BinaryExpr{
		X: &goast.CallExpr{
			Fun: goast.NewIdent("pow"),
			Args: []goast.Expr{
				x1,
				createFloat(fmt.Sprintf("%d", exn-1)),
			},
		},
		Op: token.QUO,
		Y:  &goast.ParenExpr{X: x2},
	}
}

func openParenSingleIdent(a goast.Expr, variables []string) (changed bool, r goast.Expr) {
	par, ok := a.(*goast.ParenExpr)
	if !ok {
		return false, nil
	}

	// from:
	// (number)
	// to:
	// number
	num, ok := par.X.(*goast.Ident)
	if !ok {
		return false, nil
	}

	return true, num
}

func openParenSingleNumber(a goast.Expr, variables []string) (changed bool, r goast.Expr) {
	par, ok := a.(*goast.ParenExpr)
	if !ok {
		return false, nil
	}

	// from:
	// (number)
	// to:
	// number
	num, ok := par.X.(*goast.BinaryExpr)
	if !ok {
		return false, nil
	}

	return true, num
}

func openParen(a goast.Expr, variables []string) (changed bool, r goast.Expr) {
	par, ok := a.(*goast.ParenExpr)
	if !ok {
		return false, nil
	}

	// from:
	// (... */ ...)
	// to:
	// (...) */  (...)
	bin, ok := par.X.(*goast.BinaryExpr)
	if !ok {
		return false, nil
	}
	if bin.Op != token.MUL && bin.Op != token.QUO {
		return false, nil
	}
	var (
		Op = bin.Op
		X  = bin.X
		Y  = bin.Y
	)

	switch X.(type) {
	// no need paren
	case *goast.BasicLit, *goast.Ident:

	default:
		X = &goast.ParenExpr{X: X}
	}

	switch Y.(type) {
	// no need paren
	case *goast.BasicLit, *goast.Ident:

	default:
		Y = &goast.ParenExpr{X: Y}
	}

	r = &goast.BinaryExpr{
		X:  X,
		Op: Op,
		Y:  Y,
	}

	return true, r
}

func openParenRight(a goast.Expr, variables []string) (changed bool, r goast.Expr) {
	v, ok := a.(*goast.BinaryExpr)
	if !ok {
		return false, nil
	}

	// from:
	// any * (... -+ ...)
	// to:
	// ((any * X) -+  (any * Y))
	if v.Op != token.MUL {
		return false, nil
	}
	var bin *goast.BinaryExpr
	var found bool
	if par, ok := insideParen(v.Y); ok {
		if b, ok := par.(*goast.BinaryExpr); ok {
			bin = b
			found = true
		}
	} else {
		if b, ok := v.Y.(*goast.BinaryExpr); ok {
			bin = b
			found = true
		}
	}
	if !found {
		return false, nil
	}

	// create workspace
	var (
		any = v.X
		X   = bin.X
		Y   = bin.Y
		Op  = bin.Op
	)

	{
		// try simplification inside paren
		c, b := walk(bin, variables)
		if c {
			return true, &goast.BinaryExpr{
				X:  any,
				Op: token.MUL,
				Y:  b,
			}
		}
	}

	if bin.Op != token.ADD && bin.Op != token.SUB {
		return false, nil
	}

	return true, &goast.ParenExpr{
		X: &goast.BinaryExpr{
			X: &goast.ParenExpr{X: &goast.BinaryExpr{
				X:  any,
				Op: token.MUL,
				Y:  X,
			}},
			Op: Op,
			Y: &goast.ParenExpr{X: &goast.BinaryExpr{
				X:  any,
				Op: token.MUL,
				Y:  Y,
			}},
		},
	}
}

func insideParen(a goast.Expr) (in goast.Expr, ok bool) {
	if u, ok := a.(*goast.ParenExpr); ok {
		var s goast.Expr = u
		for {
			g, ok := s.(*goast.ParenExpr)
			if !ok {
				return s, true
			}
			s = g.X
		}
	}
	return nil, false
}

func openParenLeft(a goast.Expr, variables []string) (changed bool, r goast.Expr) {
	v, ok := a.(*goast.BinaryExpr)
	if !ok {
		return false, nil
	}

	// from:
	// (...) * any
	// to:
	// any * (...)
	if v.Op != token.MUL {
		return false, nil
	}

	var found bool
	if par, ok := v.X.(*goast.ParenExpr); ok {
		if _, ok := par.X.(*goast.BinaryExpr); ok {
			found = true
		}
	} else {
		if _, ok := v.X.(*goast.BinaryExpr); ok {
			found = true
		}
	}
	if !found {
		return false, nil
	}

	v.X, v.Y = v.Y, v.X
	return true, v
}

func constantsLeft(a goast.Expr, variables []string) (changed bool, r goast.Expr) {
	v, ok := a.(*goast.BinaryExpr)
	if !ok {
		return false, nil
	}
	// any + constants
	xOk, _ := isConstant(v.X)
	yOk, _ := isConstant(v.Y)
	if !(!xOk && yOk) {
		return false, nil
	}

	switch v.Op {
	case token.ADD, // +
		token.MUL: // *

	default:
		return false, nil
	}

	// swap
	v.X, v.Y = v.Y, v.X
	return true, v
}

func constantsLeftLeft(a goast.Expr, variables []string) (changed bool, r goast.Expr) {
	v, ok := a.(*goast.BinaryExpr)
	if !ok {
		return false, nil
	}
	if v.Op != token.MUL {
		return false, nil
	}
	bin, ok := v.Y.(*goast.BinaryExpr)
	if !ok {
		return false, nil
	}
	if bin.Op != token.MUL {
		return false, nil
	}

	con, _ := isConstant(bin.X)
	if !con {
		return false, nil
	}

	// from:
	// any1 * ( constants * any2)
	// to:
	// constants * (any1 * any2)
	return true, &goast.BinaryExpr{
		X:  bin.X,
		Op: token.MUL,
		Y: &goast.BinaryExpr{
			X:  v.X,
			Op: token.MUL,
			Y:  bin.Y,
		},
	}
}

func sortIdentMul(a goast.Expr, variables []string) (changed bool, r goast.Expr) {
	v, ok := a.(*goast.BinaryExpr)
	if !ok {
		return false, nil
	}
	if v.Op != token.MUL {
		return false, nil
	}
	x, ok := v.X.(*goast.Ident)
	if !ok {
		return false, nil
	}
	y, ok := v.Y.(*goast.Ident)
	if !ok {
		return false, nil
	}
	if strings.Compare(x.Name, y.Name) <= 0 {
		return false, nil
	}

	// from :
	// (b*a)
	// to :
	// (a*b)
	return true, &goast.BinaryExpr{
		X:  y,
		Op: token.MUL,
		Y:  x,
	}
}

func constants(a goast.Expr, variables []string) (changed bool, r goast.Expr) {
	v, ok := a.(*goast.BinaryExpr)
	if !ok {
		return false, nil
	}
	// constants + constants
	xOk, x := isConstant(v.X)
	yOk, y := isConstant(v.Y)
	if !xOk || !yOk {
		return false, nil
	}
	var result float64
	switch v.Op {
	case token.ADD: // +
		result = x + y
	case token.SUB: // -
		result = x - y
	case token.MUL: // *
		result = x * y
	case token.QUO: // /
		result = x / y
	default:
		panic(v.Op)
	}

	return true, createFloat(fmt.Sprintf("%.15e", result))
}

func createFloat(value string) *goast.BasicLit {
	val, err := strconv.ParseFloat(value, 64)
	if err != nil {
		panic(err)
	}
	return &goast.BasicLit{
		Kind:  token.FLOAT,
		Value: fmt.Sprintf("%.3f", val),
	}
}

func isConstant(node goast.Node) (ok bool, val float64) {
	if x, ok := node.(*goast.BasicLit); ok {
		if x.Kind == token.INT || x.Kind == token.FLOAT {
			val, err := strconv.ParseFloat(x.Value, 64)
			if err == nil {
				return true, val
			}
			panic(err)
		}
	}
	return false, 0.0
}
