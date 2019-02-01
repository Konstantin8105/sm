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

	"github.com/Konstantin8105/errors"
)

type sm struct {
	base string
	expr string
	cons []string
	vars []string
	funs []function
	iter int64
}

func (s sm) isConstant(name string) bool {
	for i := range s.cons {
		if s.cons[i] == name {
			return true
		}
	}
	return false
}

func (s sm) isVariable(name string) bool {
	for i := range s.vars {
		if s.vars[i] == name {
			return true
		}
	}
	return false
}

func (s sm) errorGen(e error) error {
	var et errors.Tree
	et.Name = "Error of symbolic math"
	_ = et.Add(fmt.Errorf("Expression: %s", s.base))
	{
		var ei errors.Tree
		ei.Name = "Constants :"
		for i := range s.cons {
			_ = ei.Add(fmt.Errorf("%s", s.cons[i]))
		}
		_ = et.Add(ei)
	}
	{
		var ei errors.Tree
		ei.Name = "Variables :"
		for i := range s.vars {
			_ = ei.Add(fmt.Errorf("%s", s.vars[i]))
		}
		_ = et.Add(ei)
	}
	{
		var ei errors.Tree
		ei.Name = "Functions :"
		for i := range s.funs {
			_ = ei.Add(fmt.Errorf("%s %v", s.funs[i].name, s.funs[i].variables))
		}
		_ = et.Add(ei)
	}
	_ = et.Add(fmt.Errorf("Iteration : %d", s.iter))
	_ = et.Add(fmt.Errorf("Error     : %v", e))
	return et
}

func (s sm) iterationLimit() error {
	var maxIteration int64 = 1000000
	if s.iter > maxIteration {
		return s.errorGen(fmt.Errorf("iteration limit"))
	}
	return nil
}

type function struct {
	name      string
	variables []string
}

// Sexpr - simplification of expression.
func Sexpr(out io.Writer, expr string) (re string, err error) {
	if out == nil {
		out = os.Stdout
	}

	var s sm
	s.base = expr

	// split expression
	lines := strings.Split(expr, ";")
	// parse to full expression to parts
	for i := range lines {
		if strings.TrimSpace(lines[i]) == "" {
			continue
		}
		a, err := parser.ParseExpr(lines[i])
		if err != nil {
			return "", s.errorGen(err)
		}
		if call, ok := a.(*goast.CallExpr); ok {
			funIdent, ok := call.Fun.(*goast.Ident)
			if !ok {
				return "", s.errorGen(fmt.Errorf("not good function name: %s", lines[i]))
			}
			// function name
			switch funIdent.Name {
			case "function":
				if len(call.Args) < 2 {
					return "", s.errorGen(fmt.Errorf("function have minimal 2 arguments - name of function and depend variable"))
				}
				var f function
				// name of function
				if id, ok := call.Args[0].(*goast.Ident); ok {
					f.name = id.Name
				} else {
					return "", s.errorGen(fmt.Errorf("not valid name of function"))
				}
				// depend variables
				for i := 1; i < len(call.Args); i++ {
					if id, ok := call.Args[i].(*goast.Ident); ok {
						f.variables = append(f.variables, id.Name)
						s.vars = append(s.vars, id.Name)
					} else {
						return "", s.errorGen(fmt.Errorf("not valid name of variable"))
					}
				}
				s.funs = append(s.funs, f)
				continue
			case "constant":
				if len(call.Args) != 1 {
					return "", s.errorGen(fmt.Errorf("constants have only one argument - name of constant"))
				}
				if id, ok := call.Args[0].(*goast.Ident); ok {
					s.cons = append(s.cons, id.Name)
				} else {
					return "", s.errorGen(fmt.Errorf("not valid name of constant"))
				}
				continue
			case "variable":
				if len(call.Args) != 1 {
					return "", s.errorGen(fmt.Errorf("variables have only one argument - name of variable"))
				}
				if id, ok := call.Args[0].(*goast.Ident); ok {
					s.vars = append(s.vars, id.Name)
				} else {
					return "", s.errorGen(fmt.Errorf("not valid name of variable"))
				}
				continue
			}
		}
		s.base = lines[i]
	}

	// avoid extra spaces in names
	for i := range s.cons {
		s.cons[i] = strings.TrimSpace(s.cons[i])
	}
	for i := range s.vars {
		s.vars[i] = strings.TrimSpace(s.vars[i])
	}
	for i := range s.funs {
		s.funs[i].name = strings.TrimSpace(s.funs[i].name)
		for j := range s.funs[i].variables {
			s.funs[i].variables[j] = strings.TrimSpace(s.funs[i].variables[j])
		}
	}

	// parse base expression
	var a goast.Expr
	a, err = parser.ParseExpr(s.base)
	if err != nil {
		return
	}

	var changed bool
	for {
		changed, a, err = s.walk(a)
		if err != nil {
			return "", err
		}

		// debug
		fmt.Fprintf(out, "%s\n", astToStr(a))

		if !changed {
			break
		}
		if err := s.iterationLimit(); err != nil {
			return "", err
		}
		s.iter++
	}

	// debug
	// goast.Print(token.NewFileSet(), a)

	re = astToStr(a)

	return
}

func astToStr(a goast.Expr) string {
	var buf bytes.Buffer
	printer.Fprint(&buf, token.NewFileSet(), a)
	return buf.String()
}

var counter int

func (s *sm) walk(a goast.Expr) (c bool, _ goast.Expr, _ error) {
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

	// iteration limit
	if err := s.iterationLimit(); err != nil {
		return false, nil, err
	}
	s.iter++

	// try simplification
	{
		var (
			changed bool
			r       goast.Expr
			err     error
		)
	begin:
		for _, rule := range []func(goast.Expr) (bool, goast.Expr, error){
			s.constants,             // 00
			s.constantsLeft,         // 01
			s.constantsLeftLeft,     // 02
			s.openParenLeft,         // 03
			s.openParenRight,        // 04
			s.openParen,             // 05
			s.openParenSingleNumber, // 06
			s.openParenSingleIdent,  // 07
			s.sortIdentMul,          // 08
			s.functionPow,           // 09
			s.oneMul,                // 10
			s.binaryNumber,          // 11
			s.parenParen,            // 12
			s.binaryUnary,           // 13
			s.zeroValueMul,          // 14
			s.differential,          // 15
		} {
			// fmt.Println("try rules = ", i)
			c, r, err = rule(a)
			if err != nil {
				return false, nil, err
			}
			if c {
				// fmt.Println("rules = ", i)
				// fmt.Println(astToStr(a))
				a = r
				changed = true
				goto begin
			}
		}
		if changed {
			return changed, a, nil
		}
	}

	// go deeper
	switch v := a.(type) {
	case *goast.BinaryExpr:
		cX, rX, err := s.walk(v.X)
		if err != nil {
			return false, nil, err
		}
		if cX {
			v.X = rX
			return true, v, nil
		}
		cY, rY, err := s.walk(v.Y)
		if err != nil {
			return false, nil, err
		}
		if cY {
			v.Y = rY
			return true, v, nil
		}

	case *goast.ParenExpr:
		return s.walk(v.X)

	case *goast.BasicLit:
		if v.Kind == token.INT {
			return true, createFloat(v.Value), nil
		}

	case *goast.Ident: // ignore

	case *goast.UnaryExpr:
		if bas, ok := v.X.(*goast.BasicLit); ok {
			return true, createFloat(fmt.Sprintf("%v%s", v.Op, bas.Value)), nil
		}
		c, e, err := s.walk(v.X)
		if err != nil {
			return false, nil, err
		}
		if c {
			return true, &goast.UnaryExpr{
				Op: v.Op,
				X:  e,
			}, nil
		}

	case *goast.CallExpr:
		var call goast.CallExpr
		call.Fun = v.Fun
		var changed bool
		for i := range v.Args {
			c, e, err := s.walk(v.Args[i])
			if err != nil {
				return false, nil, err
			}
			if c {
				changed = true
				call.Args = append(call.Args, e)
				continue
			}
			call.Args = append(call.Args, v.Args[i])
		}
		if changed {
			return true, &call, nil
		}

	default:
		panic(fmt.Errorf("Add implementation for type %T", a))
	}

	// all is not changed
	return false, a, nil
}

func (s *sm) binaryUnary(a goast.Expr) (changed bool, r goast.Expr, _ error) {
	bin, ok := a.(*goast.BinaryExpr)
	if !ok {
		return false, nil, nil
	}
	if bin.Op != token.ADD && bin.Op != token.SUB {
		return false, nil, nil
	}

	var unary *goast.UnaryExpr
	found := false
	if par, ok := bin.Y.(*goast.ParenExpr); ok {
		if un, ok := par.X.(*goast.UnaryExpr); ok {
			unary = un
			found = true
		}
	}
	if un, ok := bin.Y.(*goast.UnaryExpr); ok {
		unary = un
		found = true
	}
	if !found {
		return false, nil, nil
	}

	// from:
	// ... + (-...)
	// to:
	// ... - (...)
	if bin.Op == token.ADD && unary.Op == token.SUB {
		return true, &goast.BinaryExpr{
			X:  bin.X,
			Op: token.SUB,
			Y:  unary.X,
		}, nil
	}

	// from:
	// ... - (-...)
	// to:
	// ... + (...)
	if bin.Op == token.SUB && unary.Op == token.SUB {
		return true, &goast.BinaryExpr{
			X:  bin.X,
			Op: token.ADD,
			Y:  unary.X,
		}, nil
	}

	// from:
	// ... - (+...)
	// to:
	// ... - (...)
	if bin.Op == token.SUB && unary.Op == token.ADD {
		return true, &goast.BinaryExpr{
			X:  bin.X,
			Op: token.SUB,
			Y:  unary.X,
		}, nil
	}

	// from:
	// ... + (+...)
	// to:
	// ... + (...)
	return true, &goast.BinaryExpr{
		X:  bin.X,
		Op: token.ADD,
		Y:  unary.X,
	}, nil
}

func (s *sm) parenParen(a goast.Expr) (changed bool, r goast.Expr, _ error) {
	par, ok := a.(*goast.ParenExpr)
	if !ok {
		return false, nil, nil
	}
	parPar, ok := par.X.(*goast.ParenExpr)
	if !ok {
		return false, nil, nil
	}

	// from :
	// (( ... ))
	// to :
	// (...)
	return true, parPar, nil
}

func (s *sm) binaryNumber(a goast.Expr) (changed bool, r goast.Expr, _ error) {
	bin, ok := a.(*goast.BinaryExpr)
	if !ok {
		return false, nil, nil
	}
	if bin.Op != token.ADD && bin.Op != token.SUB {
		return false, nil, nil
	}

	leftBin, ok := bin.Y.(*goast.BinaryExpr)
	if !ok {
		return false, nil, nil
	}
	if leftBin.Op != token.ADD && leftBin.Op != token.SUB {
		return false, nil, nil
	}

	nOk1, _ := isConstant(bin.X)
	if !nOk1 {
		return false, nil, nil
	}
	nOk2, _ := isConstant(leftBin.X)
	if !nOk2 {
		return false, nil, nil
	}

	num1 := bin.X
	num2 := leftBin.X

	// from:
	// number1 + (number2 +- ...)
	// to:
	// (number1 + number2) +- (...)
	if bin.Op == token.ADD {
		return true, &goast.BinaryExpr{
			X: &goast.ParenExpr{
				X: &goast.BinaryExpr{
					X:  num1,
					Op: bin.Op,
					Y:  num2,
				},
			},
			Op: leftBin.Op,
			Y:  &goast.ParenExpr{X: leftBin.Y},
		}, nil
	}
	// from:
	// number1 - (number2 + ...)
	// to:
	// (number1 - number2) - (...)
	if bin.Op == token.SUB && leftBin.Op == token.ADD {
		return true, &goast.BinaryExpr{
			X: &goast.ParenExpr{
				X: &goast.BinaryExpr{
					X:  num1,
					Op: bin.Op,
					Y:  num2,
				},
			},
			Op: token.SUB,
			Y:  &goast.ParenExpr{X: leftBin.Y},
		}, nil
	}
	// from:
	// number1 - (number2 - ...)
	// to:
	// (number1 - number2) + (...)
	return true, &goast.BinaryExpr{
		X: &goast.ParenExpr{
			X: &goast.BinaryExpr{
				X:  num1,
				Op: bin.Op,
				Y:  num2,
			},
		},
		Op: token.ADD,
		Y:  &goast.ParenExpr{X: leftBin.Y},
	}, nil
}

func (s *sm) oneMul(a goast.Expr) (changed bool, r goast.Expr, _ error) {
	bin, ok := a.(*goast.BinaryExpr)
	if !ok {
		return false, nil, nil
	}
	if bin.Op != token.MUL {
		return false, nil, nil
	}
	bas, ok := bin.X.(*goast.BasicLit)
	if !ok {
		return false, nil, nil
	}

	val, err := strconv.ParseFloat(bas.Value, 64)
	if err != nil {
		panic(err)
	}

	if val != float64(int64(val)) {
		return false, nil, nil
	}

	exn := int64(val)
	if exn != 1 {
		return false, nil, nil
	}

	// from :
	// 1 * any
	// to:
	// any
	return true, bin.Y, nil
}

func (s *sm) differential(a goast.Expr) (changed bool, r goast.Expr, _ error) {
	call, ok := a.(*goast.CallExpr)
	if !ok {
		return false, nil, nil
	}
	id, ok := call.Fun.(*goast.Ident)
	if !ok {
		return false, nil, nil
	}
	if id.Name != "d" {
		return false, nil, nil
	}
	if len(call.Args) != 2 {
		panic("function pow have not 2 arguments")
	}
	id, ok = call.Args[1].(*goast.Ident)
	if !ok {
		return false, nil, s.errorGen(fmt.Errorf(
			"Second argument of differential is not variable"))
	}

	dvar := id.Name
	if !s.isVariable(dvar) {
		return false, nil, s.errorGen(fmt.Errorf(
			"Second argument of differential is not initialized like variable"))
	}

	if bin, ok := call.Args[0].(*goast.BinaryExpr); ok {
		switch bin.Op {
		case token.MUL: // *
			// rule:
			// d(u*v,x) = d(u,x)*v + u*d(v,x)
			// where `u` and `v` is any
			u1 := bin.X
			u2 := bin.X
			v1 := bin.Y
			v2 := bin.Y
			return true, &goast.BinaryExpr{
				X: &goast.BinaryExpr{
					X: &goast.CallExpr{
						Fun: goast.NewIdent("d"),
						Args: []goast.Expr{
							u1,
							id,
						},
					},
					Op: token.MUL,
					Y:  v1,
				},
				Op: token.ADD,
				Y: &goast.BinaryExpr{
					X:  u2,
					Op: token.MUL,
					Y: &goast.CallExpr{
						Fun: goast.NewIdent("d"),
						Args: []goast.Expr{
							v2,
							id,
						},
					},
				},
			}, nil
		}
	}

	{
		// from:
		// d(number, x)
		// to:
		// 0.000
		num, _ := isConstant(call.Args[0])
		if num {
			return true, createFloat("0"), nil
		}
	}
	{
		// from:
		// d(x,x)
		// to:
		// 1.000
		if x, ok := call.Args[0].(*goast.Ident); ok {
			if x.Name == dvar {
				return true, createFloat("1"), nil
			}
		}
	}
	{
		// from:
		// d(constant,x)
		// to:
		// constant * d(1.000,x)
		if con, ok := call.Args[0].(*goast.Ident); ok {
			name := con.Name
			if s.isConstant(name) {
				call.Args[0] = createFloat("1")
				return true, &goast.BinaryExpr{
					X:  goast.NewIdent(name),
					Op: token.MUL,
					Y:  call,
				}, nil
			}
		}

	}

	return false, nil, nil
}

func (s *sm) functionPow(a goast.Expr) (changed bool, r goast.Expr, _ error) {
	call, ok := a.(*goast.CallExpr)
	if !ok {
		return false, nil, nil
	}
	id, ok := call.Fun.(*goast.Ident)
	if !ok {
		return false, nil, nil
	}
	if id.Name != "pow" {
		return false, nil, nil
	}
	if len(call.Args) != 2 {
		panic("function pow have not 2 arguments")
	}

	e, ok := call.Args[1].(*goast.BasicLit)
	if !ok {
		return false, nil, nil
	}

	exponent, err := strconv.ParseFloat(e.Value, 64)
	if err != nil {
		panic(err)
	}

	if exponent != float64(int64(exponent)) {
		return false, nil, nil
	}

	exn := int64(exponent)

	if exn == 0 {
		// from:
		// pow(..., 0)
		// to:
		// 1
		return true, createFloat("1"), nil
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
		}, nil
	}

	// from:
	// pow(..., -33)
	// to:
	// pow(..., -32) * 1.0 / (...)
	x1 := call.Args[0]
	x2 := call.Args[0]
	return true, &goast.BinaryExpr{
		X: &goast.CallExpr{
			Fun: goast.NewIdent("pow"),
			Args: []goast.Expr{
				x1,
				createFloat(fmt.Sprintf("%d", exn+1)),
			},
		},
		Op: token.MUL,
		Y: &goast.BinaryExpr{
			X:  createFloat("1"),
			Op: token.QUO,
			Y:  &goast.ParenExpr{X: x2},
		},
	}, nil
}

func (s *sm) openParenSingleIdent(a goast.Expr) (changed bool, r goast.Expr, _ error) {
	par, ok := a.(*goast.ParenExpr)
	if !ok {
		return false, nil, nil
	}

	// from:
	// (number)
	// to:
	// number
	num, ok := par.X.(*goast.Ident)
	if !ok {
		return false, nil, nil
	}

	return true, num, nil
}

func (s *sm) openParenSingleNumber(a goast.Expr) (changed bool, r goast.Expr, _ error) {
	par, ok := a.(*goast.ParenExpr)
	if !ok {
		return false, nil, nil
	}

	// from:
	// (number)
	// to:
	// number
	num, ok := par.X.(*goast.BasicLit)
	if !ok {
		return false, nil, nil
	}

	return true, num, nil
}

func (s *sm) openParen(a goast.Expr) (changed bool, r goast.Expr, _ error) {
	par, ok := a.(*goast.ParenExpr)
	if !ok {
		return false, nil, nil
	}

	// from:
	// (... */ ...)
	// to:
	// (...) */  (...)
	bin, ok := par.X.(*goast.BinaryExpr)
	if !ok {
		return false, nil, nil
	}
	if bin.Op != token.MUL && bin.Op != token.QUO {
		return false, nil, nil
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

	return true, r, nil
}

func (s *sm) openParenRight(a goast.Expr) (changed bool, r goast.Expr, _ error) {
	v, ok := a.(*goast.BinaryExpr)
	if !ok {
		return false, nil, nil
	}

	// from:
	// any * (... -+ ...)
	// to:
	// ((any * X) -+  (any * Y))
	if v.Op != token.MUL {
		return false, nil, nil
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
		return false, nil, nil
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
		c, b, err := s.walk(bin)
		if err != nil {
			return false, nil, err
		}
		if c {
			return true, &goast.BinaryExpr{
				X:  any,
				Op: token.MUL,
				Y:  b,
			}, nil
		}
	}

	if bin.Op != token.ADD && bin.Op != token.SUB {
		return false, nil, nil
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
	}, nil
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

func (s *sm) openParenLeft(a goast.Expr) (changed bool, r goast.Expr, _ error) {
	v, ok := a.(*goast.BinaryExpr)
	if !ok {
		return false, nil, nil
	}

	// from:
	// (...) * any
	// to:
	// any * (...)
	if v.Op != token.MUL {
		return false, nil, nil
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
		return false, nil, nil
	}

	v.X, v.Y = v.Y, v.X
	return s.openParenRight(v)
}

func (s *sm) zeroValueMul(a goast.Expr) (changed bool, r goast.Expr, _ error) {
	v, ok := a.(*goast.BinaryExpr)
	if !ok {
		return false, nil, nil
	}

	xOk, x := isConstant(v.X)
	yOk, _ := isConstant(v.Y)
	if !(xOk && !yOk) {
		return false, nil, nil
	}
	if x != float64(int64(x)) {
		return false, nil, nil
	}
	if int64(x) != 0 {
		return false, nil, nil
	}

	switch v.Op {
	case token.MUL, // *
		token.QUO: // /
		// from:
		// 0.000 * any
		// to:
		// 0.000
		return true, createFloat("0"), nil

	case token.ADD, // +
		token.SUB: // -
		// from:
		// 0.000 + any
		// to:
		// any
		return true, v.Y, nil
	}

	return false, nil, nil

}

func (s *sm) constantsLeft(a goast.Expr) (changed bool, r goast.Expr, _ error) {
	v, ok := a.(*goast.BinaryExpr)
	if !ok {
		return false, nil, nil
	}
	// any + constants
	xOk, _ := isConstant(v.X)
	yOk, _ := isConstant(v.Y)
	if !(!xOk && yOk) {
		return false, nil, nil
	}

	switch v.Op {
	case token.ADD, // +
		token.MUL: // *

	default:
		return false, nil, nil
	}

	// swap
	v.X, v.Y = v.Y, v.X
	return true, v, nil
}

func (s *sm) constantsLeftLeft(a goast.Expr) (changed bool, r goast.Expr, _ error) {
	v, ok := a.(*goast.BinaryExpr)
	if !ok {
		return false, nil, nil
	}
	if v.Op != token.MUL {
		return false, nil, nil
	}
	bin, ok := v.Y.(*goast.BinaryExpr)
	if !ok {
		return false, nil, nil
	}
	if bin.Op != token.MUL {
		return false, nil, nil
	}

	con, _ := isConstant(bin.X)
	if !con {
		return false, nil, nil
	}

	con2, _ := isConstant(v.X)
	if con2 {
		return false, nil, nil
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
	}, nil
}

func (s *sm) sortIdentMul(a goast.Expr) (changed bool, r goast.Expr, _ error) {
	v, ok := a.(*goast.BinaryExpr)
	if !ok {
		return false, nil, nil
	}
	if v.Op != token.MUL {
		return false, nil, nil
	}
	x, ok := v.X.(*goast.Ident)
	if !ok {
		return false, nil, nil
	}
	y, ok := v.Y.(*goast.Ident)
	if !ok {
		return false, nil, nil
	}
	if strings.Compare(x.Name, y.Name) <= 0 {
		return false, nil, nil
	}

	// from :
	// (b*a)
	// to :
	// (a*b)
	return true, &goast.BinaryExpr{
		X:  y,
		Op: token.MUL,
		Y:  x,
	}, nil
}

func (s *sm) constants(a goast.Expr) (changed bool, r goast.Expr, _ error) {
	v, ok := a.(*goast.BinaryExpr)
	if !ok {
		return false, nil, nil
	}
	// constants + constants
	xOk, x := isConstant(v.X)
	yOk, y := isConstant(v.Y)
	if !xOk || !yOk {
		return false, nil, nil
	}

	if int64(y) == 0 && v.Op == token.QUO {
		return false, nil, s.errorGen(fmt.Errorf("cannot divide by zero"))
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

	return true, createFloat(fmt.Sprintf("%.15e", result)), nil
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
