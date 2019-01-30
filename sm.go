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

	return
}

func walk(a goast.Expr, variables []string) (changed bool, r goast.Expr) {
	// try simplification
	rules := []func(goast.Expr) (bool, goast.Expr){
		constants,
		constantsLeft,
		openParenLeft,
		openParenRight,
	}
	for i := range rules {
		changed, r = rules[i](a)
		if changed {
			return
		}
	}

	// go deeper
	switch v := a.(type) {
	case *goast.BinaryExpr:
		cX, rX := walk(v.X, variables)
		cY, rY := walk(v.Y, variables)
		changed = cX || cY
		r = a
		v.X = rX
		v.Y = rY
		return changed, r

	case *goast.ParenExpr:
		changed, r = walk(v.X, variables)
		return

	case *goast.BasicLit:
		// ignore

	case *goast.Ident:
		// ignore

	default:
		panic(fmt.Errorf("Add implementation for type %T", a))
	}

	// all is not changed
	return false, a
}

func openParenLeft(a goast.Expr) (changed bool, r goast.Expr) {
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
	par, ok := v.Y.(*goast.ParenExpr)
	if !ok {
		return false, nil
	}
	bin, ok := par.X.(*goast.BinaryExpr)
	if !ok {
		return false, nil
	}
	if bin.Op != token.ADD && bin.Op != token.SUB {
		return false, nil
	}

	// create workspace
	var (
		any = v.X
		X   = bin.X
		Y   = bin.Y
		Op  = bin.Op
	)

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

func openParenRight(a goast.Expr) (changed bool, r goast.Expr) {
	v, ok := a.(*goast.BinaryExpr)
	if !ok {
		return false, nil
	}

	// from:
	// (... -+ ...) * any
	// to:
	// ((any * X) -+ (any * Y))
	if v.Op != token.MUL {
		return false, nil
	}
	par, ok := v.X.(*goast.ParenExpr)
	if !ok {
		return false, nil
	}
	bin, ok := par.X.(*goast.BinaryExpr)
	if !ok {
		return false, nil
	}
	if bin.Op != token.ADD && bin.Op != token.SUB {
		return false, nil
	}

	// create workspace
	var (
		any = v.Y
		X   = bin.X
		Y   = bin.Y
		Op  = bin.Op
	)

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

func constantsLeft(a goast.Expr) (changed bool, r goast.Expr) {
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

func constants(a goast.Expr) (changed bool, r goast.Expr) {
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

	return true, &goast.BasicLit{
		Kind:  token.FLOAT,
		Value: fmt.Sprintf("%.3f", result),
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