package sm

import (
	"bytes"
	"fmt"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"strconv"
	"strings"

	goast "go/ast"

	"github.com/Konstantin8105/errors"
)

const (
	pow          = "pow"
	differential = "d"
	matrix       = "matrix"
	transpose    = "transpose"
	integralName = "integral"
	injectName   = "inject"
	sinName      = "sin"
	cosName      = "cos"
	tanName      = "tan"
)

type sm struct {
	base string
	expr string
	cons []string
	vars []string
	funs []function
	iter int64
}

func (s sm) isConstant(e goast.Expr) bool {
	var name string
	if ind, ok := e.(*goast.Ident); ok {
		name = ind.Name
	} else {
		return false
	}

	for i := range s.cons {
		if s.cons[i] == name {
			return true
		}
	}

	return false
}

func (s sm) isVariable(e goast.Expr) bool {
	var name string
	if ind, ok := e.(*goast.Ident); ok {
		name = ind.Name
	} else {
		return false
	}

	for i := range s.vars {
		if s.vars[i] == name {
			return true
		}
	}

	return false
}

func (s sm) isFunction(name, arg string) bool {
	for i := range s.funs {
		if name == s.funs[i].name {
			for j := range s.funs[i].variables {
				if arg == s.funs[i].variables[j] {
					return true
				}
			}
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
// Example:
//
//	expr : "b*(2+3-1+8*a)",
//	out  : "4.000*b + 8.000*(a*b)",
//
//	expr: "d(2*pow(x,a),x);constant(a);variable(x);",
//	out:  "2.000*(a*pow(x,a - 1.000))",
//
// Keywords:
//
//	constant(a); for constants
//	variables(a); for variables
//  function(a,x,y,z,...); for function a(x,y,z)
//
func Sexpr(o io.Writer, expr string) (out string, err error) {
	if o == nil {
		var buf bytes.Buffer
		o = &buf
	}
	expr = strings.Replace(expr, "\n", "", -1)

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
					return "", s.errorGen(fmt.Errorf(
						"function have minimal 2 arguments - name of function and depend variable"))
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

		fmt.Fprintf(o, "%s\n", astToStr(a))

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

	out = astToStr(a)

	return
}

func astToStr(e goast.Expr) string {
	// goast.Print(token.NewFileSet(), e)
	var buf bytes.Buffer
	printer.Fprint(&buf, token.NewFileSet(), e)
	return buf.String()
}

var counter int

func (s *sm) walk(a goast.Expr) (c bool, _ goast.Expr, _ error) {
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
		for numRule, rule := range []func(goast.Expr) (bool, goast.Expr, error){
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
			s.divideDivide,          // 16
			s.divide,                // 17
			s.matrixMultiply,        // 18
			s.matrixTranspose,       // 19
			s.mulConstToMatrix,      // 20
			s.integral,              // 21
			s.inject,                // 22
		} {
			c, r, err = rule(a)
			if err != nil {
				return false, nil, err
			}
			if c {
				_ = numRule
				// debug
				//	fmt.Printf("rules = %3d\tfrom: `%s` to `%s`\n",
				//		numRule, astToStr(a), astToStr(r))
				a = r
				changed = true
				break
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

func (s *sm) inject(e goast.Expr) (changed bool, r goast.Expr, _ error) {
	call, ok := e.(*goast.CallExpr)
	if !ok {
		return false, nil, nil
	}
	id, ok := call.Fun.(*goast.Ident)
	if !ok {
		return false, nil, nil
	}
	if id.Name != injectName {
		return false, nil, nil
	}
	//
	// from:
	// inject(x*x/2.000, x, 1.000)
	// to:
	// (1.000*1.000/2.000)
	//
	body := astToStr(call.Args[0])
	vars := astToStr(call.Args[1])
	data := astToStr(call.Args[2])

	a, err := parser.ParseExpr(strings.Replace(body, vars, data, -1))
	if err != nil {
		return false, nil, err
	}

	return true, &goast.ParenExpr{X: a}, nil
}

func (s *sm) matrixTranspose(e goast.Expr) (changed bool, r goast.Expr, _ error) {
	call, ok := e.(*goast.CallExpr)
	if !ok {
		return false, nil, nil
	}
	id, ok := call.Fun.(*goast.Ident)
	if !ok {
		return false, nil, nil
	}
	if id.Name != transpose {
		return false, nil, nil
	}
	id.Name = matrix
	mt, ok := isMatrix(call.Args[0])
	if !ok {
		panic("not valid transpose matrix")
	}
	// transpose
	var trans m
	trans.args = make([]goast.Expr, len(mt.args))
	trans.rows = mt.columns
	trans.columns = mt.rows
	for r := 0; r < mt.rows; r++ {
		for c := 0; c < mt.columns; c++ {
			trans.args[trans.position(c, r)] = mt.args[mt.position(r, c)]
		}
	}

	result := &goast.CallExpr{
		Fun:  goast.NewIdent(matrix),
		Args: trans.args,
	}
	// rows
	result.Args = append(result.Args, createFloat(fmt.Sprintf("%d", trans.rows)))
	// columns
	result.Args = append(result.Args, createFloat(fmt.Sprintf("%d", trans.columns)))

	return true, result, nil
}

func (s *sm) matrixMultiply(a goast.Expr) (changed bool, r goast.Expr, _ error) {
	// matrix(...)*matrix(...)
	bin, ok := a.(*goast.BinaryExpr)
	if !ok {
		return false, nil, nil
	}
	if bin.Op != token.MUL {
		return false, nil, nil
	}

	left, ok := isMatrix(bin.X)
	if !ok {
		return false, nil, nil
	}
	right, ok := isMatrix(bin.Y)
	if !ok {
		return false, nil, nil
	}
	if left.columns != right.rows {
		return false, nil, fmt.Errorf("not valid matrix multiplication")
	}

	result := &goast.CallExpr{
		Fun: goast.NewIdent(matrix),
	}
	// multiplication
	for lr := 0; lr < left.rows; lr++ {
		for rc := 0; rc < right.columns; rc++ {
			var arg goast.Expr
			for p := 0; p < left.columns; p++ {
				mul := &goast.BinaryExpr{
					X:  left.args[left.position(lr, p)],   // left
					Op: token.MUL,                         // *
					Y:  right.args[right.position(p, rc)], // right
				}
				// fmt.Println(left,len(left.args),	left.position(lr,p))
				if p == 0 {
					arg = mul
				} else {
					arg = &goast.BinaryExpr{
						X:  arg,
						Op: token.ADD, // +
						Y:  mul,
					}
				}
			}
			result.Args = append(result.Args, arg)
		}
	}
	// rows
	result.Args = append(result.Args, createFloat(fmt.Sprintf("%d", left.rows)))
	// columns
	result.Args = append(result.Args, createFloat(fmt.Sprintf("%d", right.columns)))

	return true, result, nil
}

func (s *sm) divideDivide(a goast.Expr) (changed bool, r goast.Expr, _ error) {
	bin, ok := a.(*goast.BinaryExpr)
	if !ok {
		return false, nil, nil
	}
	if bin.Op != token.QUO {
		return false, nil, nil
	}
	leftBin, ok := bin.X.(*goast.BinaryExpr)
	if !ok {
		return false, nil, nil
	}
	if leftBin.Op != token.QUO {
		return false, nil, nil
	}
	rightBin, ok := bin.Y.(*goast.BinaryExpr)
	if !ok {
		return false, nil, nil
	}
	if rightBin.Op != token.QUO {
		return false, nil, nil
	}

	// from:
	// (a/b)/(c/d)
	// to:
	// (a*d)/(b*c)
	return true, &goast.BinaryExpr{
		X: &goast.ParenExpr{X: &goast.BinaryExpr{
			X:  leftBin.X,
			Op: token.MUL,
			Y:  rightBin.Y,
		}},
		Op: token.QUO,
		Y: &goast.ParenExpr{X: &goast.BinaryExpr{
			X:  leftBin.Y,
			Op: token.MUL,
			Y:  rightBin.X,
		}},
	}, nil
}

func (s *sm) divide(a goast.Expr) (changed bool, r goast.Expr, _ error) {
	bin, ok := a.(*goast.BinaryExpr)
	if !ok {
		return false, nil, nil
	}
	if bin.Op != token.QUO {
		return false, nil, nil
	}
	leftBin, ok := bin.Y.(*goast.BinaryExpr)
	if !ok {
		return false, nil, nil
	}
	if leftBin.Op != token.QUO {
		return false, nil, nil
	}

	// from:
	// a/(b/c)
	// to:
	// (a*b)/c
	return true, &goast.BinaryExpr{
		X: &goast.BinaryExpr{
			X:  bin.X,
			Op: token.MUL,
			Y:  leftBin.Y,
		},
		Op: token.QUO,
		Y:  leftBin.X,
	}, nil
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
	if bin, ok := par.X.(*goast.BinaryExpr); ok {
		return true, bin, nil
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
	leftBin, ok := bin.Y.(*goast.BinaryExpr)
	if !ok {
		return false, nil, nil
	}

	iscon := func(e goast.Expr) bool {
		var ok1, ok2 bool
		ok1, _ = isConstant(e)
		// 		if ind, ok := e.(*goast.Ident); ok {
		// 			ok2 = s.isConstant(ind.Name)
		// 		}
		if ok1 || ok2 {
			return true
		}
		return false
	}
	num1, num2 := bin.X, leftBin.X

	if ok := iscon(num1); !ok {
		return false, nil, nil
	}
	if ok := iscon(num2); !ok {
		return false, nil, nil
	}

	//
	// from:
	// number1 * (number2 / ...)
	// to:
	// (number1 * number2) / (...)
	//
	// from:
	// number1 * (number2 * ...)
	// to:
	// (number1 * number2) * (...)
	//
	if bin.Op == token.MUL && (leftBin.Op == token.QUO || leftBin.Op == token.MUL) {
		return true, &goast.BinaryExpr{
			X: &goast.ParenExpr{
				X: &goast.BinaryExpr{
					X:  num1,
					Op: token.MUL,
					Y:  num2,
				},
			},
			Op: leftBin.Op,
			Y:  &goast.ParenExpr{X: leftBin.Y},
		}, nil
	}

	if bin.Op != token.ADD && bin.Op != token.SUB {
		return false, nil, nil
	}
	if leftBin.Op != token.ADD && leftBin.Op != token.SUB {
		return false, nil, nil
	}

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
	if id.Name != differential {
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
	if !s.isVariable(id) {
		return false, nil, s.errorGen(fmt.Errorf(
			"Second argument of differential is not initialized like variable"+
				": `%s`", dvar))
	}

	if bin, ok := call.Args[0].(*goast.BinaryExpr); ok {
		switch bin.Op {
		case token.ADD, token.SUB: // + -
			// rule:
			// d(u + v, x) = d(u,x) + d(v,x)
			return true, &goast.BinaryExpr{
				X: &goast.CallExpr{
					Fun: goast.NewIdent(differential),
					Args: []goast.Expr{
						bin.X,
						goast.NewIdent(dvar),
					},
				},
				Op: bin.Op,
				Y: &goast.CallExpr{
					Fun: goast.NewIdent(differential),
					Args: []goast.Expr{
						bin.Y,
						goast.NewIdent(dvar),
					},
				},
			}, nil

		case token.QUO: // /
			// rule:
			// d(u/v,x) = (d(u,x)*v - u*d(v,x)) / (v * v)
			// where `u` and `v` is any
			u1 := bin.X
			u2 := bin.X
			v1 := bin.Y
			v2 := bin.Y
			v3 := bin.Y
			return true, &goast.BinaryExpr{
				X: &goast.ParenExpr{X: &goast.BinaryExpr{
					X: &goast.BinaryExpr{
						X: &goast.CallExpr{
							Fun: goast.NewIdent(differential),
							Args: []goast.Expr{
								u1,
								goast.NewIdent(dvar),
							},
						},
						Op: token.MUL,
						Y:  v1,
					},
					Op: token.SUB,
					Y: &goast.BinaryExpr{
						X:  u2,
						Op: token.MUL,
						Y: &goast.CallExpr{
							Fun: goast.NewIdent(differential),
							Args: []goast.Expr{
								v2,
								goast.NewIdent(dvar),
							},
						},
					},
				}},
				Op: token.QUO,
				Y: &goast.ParenExpr{X: &goast.BinaryExpr{
					X:  v3,
					Op: token.MUL,
					Y:  v3,
				}},
			}, nil

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
						Fun: goast.NewIdent(differential),
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
						Fun: goast.NewIdent(differential),
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
		val, exp, ok, err := isFunctionPow(call.Args[0])
		if ok {
			if err != nil {
				return false, nil, s.errorGen(err)
			}
			if x, ok := val.(*goast.Ident); ok && x.Name == dvar {
				found := false
				if ok := s.isConstant(exp); ok {
					found = true
				}
				if ok, _ := isConstant(exp); ok {
					found = true
				}

				if found {
					// from:
					// d(pow(x,a), x)
					// where a is constant or number
					// to:
					// a * pow(x, a-1)
					return true, &goast.BinaryExpr{
						X:  exp,
						Op: token.MUL,
						Y: &goast.CallExpr{
							Fun: goast.NewIdent(pow),
							Args: []goast.Expr{
								goast.NewIdent(dvar),
								&goast.BinaryExpr{
									X:  exp,
									Op: token.SUB,
									Y:  createFloat("1.000"),
								},
							},
						},
					}, nil
				}
			}
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
		if s.isConstant(call.Args[0]) {
			call.Args[0] = createFloat("1")
			return true, &goast.BinaryExpr{
				X:  call.Args[0],
				Op: token.MUL,
				Y:  call,
			}, nil
		}
	}
	{
		// from :
		// d(a,x); function(a,z);
		// to:
		// 0.000
		if id, ok := call.Args[0].(*goast.Ident); ok {
			if ok := s.isFunction(id.Name, dvar); !ok {
				return true, createFloat("0.0"), nil
			}
		}
	}

	return false, nil, nil
}

func isFunctionPow(a goast.Expr) (val, exp goast.Expr, ok bool, err error) {
	call, ok := a.(*goast.CallExpr)
	if !ok {
		return nil, nil, false, nil
	}
	id, ok := call.Fun.(*goast.Ident)
	if !ok {
		return nil, nil, false, nil
	}
	if id.Name != pow {
		return nil, nil, false, nil
	}
	if len(call.Args) != 2 {
		return nil, nil, true, fmt.Errorf("function pow have not 2 arguments")
	}

	return call.Args[0], call.Args[1], true, nil
}

func (s *sm) functionPow(a goast.Expr) (changed bool, r goast.Expr, _ error) {
	val, exp, ok, err := isFunctionPow(a)
	if !ok {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, s.errorGen(err)
	}

	e, ok := exp.(*goast.BasicLit)
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
		x1 := val
		x2 := val
		return true, &goast.BinaryExpr{
			X:  &goast.ParenExpr{X: x1},
			Op: token.MUL,
			Y: &goast.CallExpr{
				Fun: goast.NewIdent(pow),
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
	x1 := val
	x2 := val
	return true, &goast.BinaryExpr{
		X: &goast.CallExpr{
			Fun: goast.NewIdent(pow),
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
	if v.Op == token.MUL {
		if _, ok := v.X.(*goast.BinaryExpr); ok {
			if _, ok := v.Y.(*goast.Ident); ok {
				v.X, v.Y = v.Y, v.X // swap
				return true, &goast.BinaryExpr{
					X:  v.X,
					Op: token.MUL,
					Y:  v.Y,
				}, nil
			}
		}
	}

	// from:
	// (...) / any
	// to:
	// 1.000/any * (...)
	if v.Op == token.QUO {
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
		found = false
		if ok, _ := isConstant(v.Y); ok {
			found = true
		}
		if ok := s.isConstant(v.Y); ok {
			found = true
		}

		if found {
			return true, &goast.BinaryExpr{
				X: &goast.BinaryExpr{
					X:  createFloat("1"),
					Op: token.QUO,
					Y:  v.Y,
				},
				Op: token.MUL,
				Y:  v.X,
			}, nil
		}
	}

	return false, nil, nil
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
	case token.ADD, token.MUL: // + , *
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

func (s *sm) integral(e goast.Expr) (changed bool, r goast.Expr, _ error) {
	call, ok := e.(*goast.CallExpr)
	if !ok {
		return false, nil, nil
	}
	id, ok := call.Fun.(*goast.Ident)
	if !ok {
		return false, nil, nil
	}
	if id.Name != integralName {
		return false, nil, nil
	}

	ok, from := isConstant(call.Args[len(call.Args)-2])
	if !ok {
		return false, nil, nil
	}
	ok, to := isConstant(call.Args[len(call.Args)-1])
	if !ok {
		return false, nil, nil
	}
	args := call.Args[:len(call.Args)-3]
	vars := call.Args[len(call.Args)-3]

	if len(args) != 1 {
		panic(fmt.Errorf("strange len of intergal: %d", len(args)))
	}
	ifn := args[0]

	if ok, v := isConstant(ifn); ok {
		switch v {
		case 0.0:
			return true, createFloat("0.000"), nil

		case 1.0:
			return true, &goast.ParenExpr{
				X: &goast.BinaryExpr{
					X:  createFloat(fmt.Sprintf("%15e", to)),
					Op: token.SUB, // -
					Y:  createFloat(fmt.Sprintf("%15e", from)),
				},
			}, nil

		default:
			return true, &goast.BinaryExpr{
				X:  createFloat(fmt.Sprintf("%15e", v)),
				Op: token.MUL,
				Y: &goast.CallExpr{
					Fun: goast.NewIdent(integralName),
					Args: []goast.Expr{
						createFloat(fmt.Sprintf("%15e", 1.0)),
						vars,
						createFloat(fmt.Sprintf("%15e", from)),
						createFloat(fmt.Sprintf("%15e", to)),
					},
				},
			}, nil
		}
	}

	// integral(a,...)
	if ok := s.isConstant(ifn); ok {
		return true, &goast.BinaryExpr{
			X:  ifn,
			Op: token.MUL,
			Y: &goast.CallExpr{
				Fun: goast.NewIdent(integralName),
				Args: []goast.Expr{
					createFloat(fmt.Sprintf("%15e", 1.0)),
					vars,
					createFloat(fmt.Sprintf("%15e", from)),
					createFloat(fmt.Sprintf("%15e", to)),
				},
			},
		}, nil
	}

	// integral(...+...)
	// integral(...)+integral(...)
	if bin, ok := ifn.(*goast.BinaryExpr); ok {
		if bin.Op == token.ADD || bin.Op == token.SUB {
			return true, &goast.BinaryExpr{
				X: &goast.CallExpr{
					Fun: goast.NewIdent(integralName),
					Args: []goast.Expr{
						bin.X,
						vars,
						createFloat(fmt.Sprintf("%15e", from)),
						createFloat(fmt.Sprintf("%15e", to)),
					},
				},
				Op: bin.Op,
				Y: &goast.CallExpr{
					Fun: goast.NewIdent(integralName),
					Args: []goast.Expr{
						bin.Y,
						vars,
						createFloat(fmt.Sprintf("%15e", from)),
						createFloat(fmt.Sprintf("%15e", to)),
					},
				},
			}, nil
		}
	}

	if bin, ok := ifn.(*goast.BinaryExpr); ok {
		if bin.Op == token.MUL {
			left := bin.X
			right := bin.Y
			// from:
			// integral(a * ...)
			// to:
			// a*integral(...)
			if ok := s.isConstant(left); ok {
				return true, &goast.BinaryExpr{
					X:  left,
					Op: token.MUL,
					Y: &goast.CallExpr{
						Fun: goast.NewIdent(integralName),
						Args: []goast.Expr{
							right,
							vars,
							createFloat(fmt.Sprintf("%15e", from)),
							createFloat(fmt.Sprintf("%15e", to)),
						},
					},
				}, nil
			}
			// from:
			// integral(1.000/a * ...)
			// to:
			// 1.000/a*integral(...)
			if leftBin, ok := left.(*goast.BinaryExpr); ok && leftBin.Op == token.QUO {
				if ok, _ := isConstant(leftBin.X); ok {
					if ok := s.isConstant(leftBin.Y); ok {
						return true, &goast.BinaryExpr{
							X:  left,
							Op: token.MUL,
							Y: &goast.CallExpr{
								Fun: goast.NewIdent(integralName),
								Args: []goast.Expr{
									right,
									vars,
									createFloat(fmt.Sprintf("%15e", from)),
									createFloat(fmt.Sprintf("%15e", to)),
								},
							},
						}, nil
					}
				}
			}
		}
		if bin.Op == token.QUO {
			// from:
			// integral(1.000/a, x,...)
			// to:
			// 1.000/a*integral(1.000, x, ...)
			left := bin.X
			right := bin.Y
			if ok, _ := isConstant(left); ok {
				if ok := s.isConstant(right); ok {
					return true, &goast.BinaryExpr{
						X:  bin,
						Op: token.MUL,
						Y: &goast.CallExpr{
							Fun: goast.NewIdent(integralName),
							Args: []goast.Expr{
								createFloat(fmt.Sprintf("%15e", 1.000)),
								vars,
								createFloat(fmt.Sprintf("%15e", from)),
								createFloat(fmt.Sprintf("%15e", to)),
							},
						},
					}, nil
				}
			}
		}
	}

	//
	// D(pow(x,n+1)/(n+1), 0.000, 1.000)
	//
	// n = 1
	// integral(x, x, 0.000, 1.000)
	// inject(pow(x,1+1)/(1+1), x, 1) - inject(pow(x,1+1)/(1+1), x, 0)
	//
	{
		body := astToStr(ifn)
		n := float64(strings.Count(body, astToStr(vars)))
		body = strings.Replace(body, astToStr(vars), "", -1)
		body = strings.Replace(body, "(", "", -1)
		body = strings.Replace(body, ")", "", -1)
		body = strings.Replace(body, "*", "", -1)
		body = strings.TrimSpace(body)

		if body == "" {
			power := &goast.CallExpr{
				Fun: goast.NewIdent("pow"),
				Args: []goast.Expr{
					vars,
					createFloat(fmt.Sprintf("%15e", n+1.0)),
				},
			}
			div := &goast.BinaryExpr{
				X:  power,
				Op: token.QUO,
				Y:  createFloat(fmt.Sprintf("%15e", n+1.0)),
			}
			return true, &goast.ParenExpr{
				X: &goast.BinaryExpr{
					X: &goast.CallExpr{
						Fun: goast.NewIdent("inject"),
						Args: []goast.Expr{
							div,
							vars,
							createFloat(fmt.Sprintf("%15e", to)),
						},
					},
					Op: token.SUB,
					Y: &goast.CallExpr{
						Fun: goast.NewIdent("inject"),
						Args: []goast.Expr{
							div,
							vars,
							createFloat(fmt.Sprintf("%15e", from)),
						},
					},
				},
			}, nil
		}
	}

	// integral(matrix(...),x,0,1)
	if len(args) == 1 {
		if mt, ok := isMatrix(args[0]); ok {
			// TODO
			for i := 0; i < len(mt.args); i++ {
				mt.args[i] = &goast.CallExpr{
					Fun: goast.NewIdent(integralName),
					Args: []goast.Expr{
						&goast.ParenExpr{X: mt.args[i]},
						vars,
						createFloat(fmt.Sprintf("%15e", from)),
						createFloat(fmt.Sprintf("%15e", to)),
					},
				}
			}
			result := &goast.CallExpr{
				Fun:  goast.NewIdent(matrix),
				Args: mt.args,
			}
			// rows
			result.Args = append(result.Args, createFloat(fmt.Sprintf("%d", mt.rows)))
			// columns
			result.Args = append(result.Args, createFloat(fmt.Sprintf("%d", mt.columns)))

			return true, result, nil
		}
	}

	return false, nil, nil
}

func (s *sm) mulConstToMatrix(a goast.Expr) (changed bool, r goast.Expr, _ error) {
	v, ok := a.(*goast.BinaryExpr)
	if !ok {
		return false, nil, nil
	}
	if v.Op != token.MUL {
		return false, nil, nil
	}
	_, ok = isMatrix(v.X)
	if ok {
		return false, nil, nil
	}
	ok = isTranspose(v.X)
	if ok {
		return false, nil, nil
	}
	mt, ok := isMatrix(v.Y)
	if !ok {
		return false, nil, nil
	}

	for i := 0; i < len(mt.args); i++ {
		mt.args[i] = &goast.BinaryExpr{
			X:  &goast.ParenExpr{X: mt.args[i]},
			Op: token.MUL, // *
			Y:  &goast.ParenExpr{X: v.X},
		}
	}

	result := &goast.CallExpr{
		Fun:  goast.NewIdent(matrix),
		Args: mt.args,
	}
	// rows
	result.Args = append(result.Args, createFloat(fmt.Sprintf("%d", mt.rows)))
	// columns
	result.Args = append(result.Args, createFloat(fmt.Sprintf("%d", mt.columns)))

	return true, result, nil
}

func (s *sm) swap(left, right goast.Expr) bool {
	// sort priority
	//	constant
	//	function
	//	matrix
	var swap bool

	for _, fb := range []func() bool{
		func() (b bool) {
			if ok, _ := isConstant(left); ok {
				return
			}
			if ok, _ := isConstant(right); !ok {
				return
			}
			return
		},
		func() (b bool) {
			x, ok := left.(*goast.Ident)
			if !ok {
				return
			}
			y, ok := right.(*goast.Ident)
			if !ok {
				return
			}
			if 0 < strings.Compare(x.Name, y.Name) {
				b = true
			}
			return
		},
		func() (b bool) {
			if _, ok := isMatrix(left); !ok {
				return
			}
			if _, ok := right.(*goast.Ident); !ok {
				return
			}
			return true
		},
	} {
		swap = swap || fb()
	}

	//fmt.Printf(	">> %s %s %#v %#v %v\n", astToStr(left), astToStr(right) , left ,right,swap)

	return swap
}

func (s *sm) sortIdentMul(a goast.Expr) (changed bool, r goast.Expr, _ error) {
	main, ok := a.(*goast.BinaryExpr)
	if !ok {
		return false, nil, nil
	}
	if main.Op != token.MUL {
		return false, nil, nil
	}

	if s.swap(main.X, main.Y) {
		// from :
		// (b*a)
		// to :
		// (a*b)
		return true, &goast.BinaryExpr{
			X:  main.Y,
			Op: token.MUL,
			Y:  main.X,
		}, nil
	}

	fok := func(e goast.Expr) bool {
		if ok, _ := isConstant(e); ok {
			return true
		}
		if ok := s.isConstant(e); ok {
			return true
		}
		return false
	}

	if left, ok := main.X.(*goast.BinaryExpr); ok && left.Op == token.MUL {
		if right, ok := main.Y.(*goast.BinaryExpr); ok && right.Op == token.MUL {
			//
			// from:
			//  left         right   //
			// ( a * x ) * ( a * x ) //
			// to:
			// ( a * a ) * ( x * x ) //
			//
			var (
				okLL = fok(left.X)
				okLR = fok(left.Y)
				okRL = fok(right.X)
			)
			if okLL && !okLR && okRL {
				left.Y, right.X = right.X, left.Y
			}
		}
	}
	// from:
	// x*(a*(...))
	// to:
	// a*(x*(...))
	if !fok(main.X) {
		if right, ok := main.Y.(*goast.BinaryExpr); ok && right.Op == token.MUL {
			if fok(right.X) {
				main.X, right.X = right.X, main.X
				return true, main, nil
			}
		}
	}

	if left, ok := main.X.(*goast.BinaryExpr); ok && left.Op == token.MUL {
		if right, ok := main.Y.(*goast.BinaryExpr); ok && right.Op == token.MUL {
			//
			// from:
			//  left         right    //
			// ( a * x ) * ( a * x )  //
			// to:
			// (a * ( x * ( a * x ))) //
			//
			return true, &goast.ParenExpr{
				X: &goast.BinaryExpr{
					X:  left.X,
					Op: token.MUL,
					Y: &goast.BinaryExpr{
						X:  left.Y,
						Op: token.MUL,
						Y:  right,
					},
				},
			}, nil
		}
	}

	return false, nil, nil
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

func createFloat(value interface{}) *goast.BasicLit {
	switch v := value.(type) {
	case float64:
		return &goast.BasicLit{
			Kind:  token.FLOAT,
			Value: fmt.Sprintf("%.3f", v),
		}
	case int:
		return createFloat(float64(v))
	case string:
		val, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			panic(fmt.Errorf("`%s` : %v", value, err))
		}
		return createFloat(val)
	}
	panic(fmt.Errorf("createFloat: %#v", value))
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

type m struct {
	args          []goast.Expr
	rows, columns int
}

func (matrix m) position(r, c int) int {
	return c + matrix.columns*r
}

func isMatrix(e goast.Expr) (mt *m, ok bool) {
	call, ok := e.(*goast.CallExpr)
	if !ok {
		return nil, false
	}
	id, ok := call.Fun.(*goast.Ident)
	if !ok {
		return nil, false
	}
	if id.Name != matrix {
		return nil, false
	}
	mt = new(m)
	if len(call.Args) < 2 {
		panic(fmt.Errorf("matrix is not valid: %#v\n%s", call, astToStr(call)))
	}
	mt.args = call.Args[:len(call.Args)-2]
	// parse rows and columns
	ok, v := isConstant(call.Args[len(call.Args)-2])
	if !ok {
		return nil, false
	}
	mt.rows = int(v)

	ok, v = isConstant(call.Args[len(call.Args)-1])
	if !ok {
		return nil, false
	}
	mt.columns = int(v)

	if len(mt.args) != mt.rows*mt.columns {
		panic(fmt.Errorf("not valid matrix: args=%d rows=%d columns=%d",
			len(mt.args),
			mt.rows,
			mt.columns,
		))
	}
	return mt, true
}

func isTranspose(e goast.Expr) (ok bool) {
	call, ok := e.(*goast.CallExpr)
	if !ok {
		return false
	}
	id, ok := call.Fun.(*goast.Ident)
	if !ok {
		return false
	}
	if id.Name != transpose {
		return false
	}
	return true
}

func debug(e goast.Expr) {
	fmt.Println("-------------")
	fmt.Println(astToStr(e))
	goast.Print(token.NewFileSet(), e)
	fmt.Println("-------------")
}
