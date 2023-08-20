package sm

import (
	"bytes"
	"container/list"
	"fmt"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"

	goast "go/ast"

	"github.com/Konstantin8105/errors"
)

const (
	pow          = "pow"
	differential = "d"
	matrix       = "matrix"
	transpose    = "transpose"
	det          = "det"
	integralName = "integral"
	injectName   = "inject"
	inverse      = "inverse"
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
	out  io.Writer
}

func (s sm) copy() (c sm) {
	c.base = s.base
	c.expr = s.expr
	c.cons = append([]string{}, s.cons...)
	c.vars = append([]string{}, s.vars...)
	c.funs = append([]function{}, s.funs...)
	c.out = s.out
	return
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
	isFunc, isVars := false, false
	for i := range s.funs {
		if s.funs[i].name == name {
			isFunc = true
		}
	}
	for i := range s.vars {
		if s.vars[i] == name {
			isVars = true
		}
	}
	if !isFunc && !isVars {
		return true
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

var MaxIteration int64 = 1000000

func (s sm) iterationLimit() error {
	if MaxIteration < 0 {
		s.iter = 0
		return nil
	}
	if MaxIteration < s.iter {
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
//		constant(a); for constants
//		variables(a); for variables
//	 function(a,x,y,z,...); for function a(x,y,z)
func Sexpr(o io.Writer, expr string) (out string, err error) {
	if o == nil {
		var buf bytes.Buffer
		o = &buf
	}
	expr = strings.Replace(expr, "\n", "", -1)

	var s sm
	s.base = expr
	s.out = o

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
				for i := range call.Args {
					if id, ok := call.Args[i].(*goast.Ident); ok {
						s.cons = append(s.cons, id.Name)
					} else {
						return "", s.errorGen(fmt.Errorf("not valid name of constant"))
					}
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

	// TODO : replace numbers(ints or floats) to constants and replace constant operations at last moment

	return s.run()
}

func (s *sm) run() (out string, err error) {
	// parse base expression
	var a goast.Expr
	a, err = parser.ParseExpr(s.base)
	if err != nil {
		return
	}

	l := list.New()
	var changed bool
	var k goast.Expr
	repeat, repeatMax := 0, 10
	for {
		// remove parens
		a, err = s.clean(a)
		if err != nil {
			return "", err
		}

		// ierarhy binary multiplication
		a, err = s.mulbin(a)
		if err != nil {
			return "", err
		}

		changed, k, err = s.walk(a)
		if err != nil {
			return "", err
		}

		str := AstToStr(k)
		s.base = str
		fmt.Fprintf(s.out, "%s\n", str)
		if changed {
			for e := l.Front(); e != nil; e = e.Next() {
				listStr := e.Value.(string)
				if listStr == str {
					repeat++
					if repeatMax < repeat {
						return "", fmt.Errorf("Repeat result: %s", str)
					}
				}
			}
			a = k
		}
		l.PushBack(str)

		if !changed {
			break
		}
		if err := s.iterationLimit(); err != nil {
			return "", err
		}

		s.iter++
	}

	out = AstToStr(a)

	return
}

func (s *sm) deeper(a goast.Expr, walker func(goast.Expr) (bool, goast.Expr, error)) (c bool, _ goast.Expr, _ error) {

	switch v := a.(type) {
	case *goast.BinaryExpr:
		cX, rX, err := walker(v.X)
		if err != nil {
			return false, nil, err
		}
		if cX {
			v.X = rX
			return true, v, nil
		}
		cY, rY, err := walker(v.Y)
		if err != nil {
			return false, nil, err
		}
		if cY {
			v.Y = rY
			return true, v, nil
		}

	case *goast.ParenExpr:
		return walker(v.X)

	case *goast.BasicLit:
		if v.Kind == token.INT {
			return true, CreateFloat(v.Value), nil
		}

	case *goast.Ident: // ignore

	case *goast.UnaryExpr:
		if bas, ok := v.X.(*goast.BasicLit); ok {
			return true, CreateFloat(fmt.Sprintf("%v%s", v.Op, bas.Value)), nil
		}
		c, e, err := walker(v.X)
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
			c, e, err := walker(v.Args[i])
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
	return false, nil, nil
}

// AstToStr convert golang ast expression to string
func AstToStr(e goast.Expr) string {
	var buf bytes.Buffer
	printer.Fprint(&buf, token.NewFileSet(), e)
	return buf.String()
}

func (s *sm) clean(a goast.Expr) (result goast.Expr, err error) {
	var changed bool
	var paren func(exp goast.Expr) (bool, goast.Expr, error)
	paren = func(exp goast.Expr) (bool, goast.Expr, error) {
		if v, ok := exp.(*goast.ParenExpr); ok {
			return true, v.X, nil
		}
		if _, ok := exp.(*goast.BasicLit); ok {
			return false, nil, nil
		}
		if _, ok := exp.(*goast.Ident); ok {
			return false, nil, nil
		}
		return s.deeper(exp, paren)
	}
	for {
		changed, result, err = s.deeper(a, paren)
		if changed {
			a = result
		} else {
			break
		}
	}
	result = a
	return
}

func (s *sm) mulbin(a goast.Expr) (result goast.Expr, err error) {
	// from : (a * b) * c
	// to   : a * (b * c)
	var changed bool
	var paren func(exp goast.Expr) (bool, goast.Expr, error)
	paren = func(exp goast.Expr) (bool, goast.Expr, error) {
		if bin, ok := exp.(*goast.BinaryExpr); ok && bin.Op == token.MUL {
			if left, ok := bin.X.(*goast.BinaryExpr); ok && left.Op == token.MUL {
				return true, &goast.BinaryExpr{
					X:  left.X,
					Op: token.MUL,
					Y: &goast.BinaryExpr{
						X:  left.Y,
						Op: token.MUL,
						Y:  bin.Y,
					},
				}, nil
			}
		}
		if _, ok := exp.(*goast.BasicLit); ok {
			return false, nil, nil
		}
		if _, ok := exp.(*goast.Ident); ok {
			return false, nil, nil
		}
		return s.deeper(exp, paren)
	}
	for {
		changed, result, err = s.deeper(a, paren)
		if changed {
			a = result
		} else {
			break
		}
	}
	result = a
	return
}

func (s *sm) walk(a goast.Expr) (c bool, result goast.Expr, _ error) {
	// iteration limit
	if err := s.iterationLimit(); err != nil {
		return false, nil, err
	}
	s.iter++

	for numRule, rule := range []func(goast.Expr) (bool, goast.Expr, error){
		func(a goast.Expr) (bool, goast.Expr, error) {
			return s.deeper(a, s.walk)
		},
		s.constants,
		s.openParen,
		s.insideParen,
		s.sort,
		s.functionPow,
		s.oneMul,
		s.divide,
		s.binaryNumber,
		s.zeroValueMul,
		s.matrixTranspose,
		s.matrixDet,
		s.matrixInverse,
		s.matrixMultiply,
		s.matrixSum,
		s.mulConstToMatrix,
		s.differential,
		s.integral,
		s.inject,
	} {
		changed, r, err := rule(a)
		if err != nil {
			return false, a, err
		}
		if changed {
			_ = numRule
			_ = os.Stdout
			// if numRule != 0 {
			//    fmt.Fprintf(os.Stdout, "> rule = %d\n", numRule)
			//    fmt.Fprintf(os.Stdout, "> from: %s --->to----> %s\n", AstToStr(a), AstToStr(r))
			// }
			a, err = parser.ParseExpr(AstToStr(r))
			if err != nil {
				return
			}
			return true, a, err
		}
	}

	return false, nil, nil
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
	if m, ok := isMatrix(call.Args[0]); ok {
		for i := range m.Args {
			m.Args[i] = &goast.CallExpr{
				Fun: goast.NewIdent(injectName),
				Args: []goast.Expr{
					m.Args[i],
					call.Args[1],
					call.Args[2],
				},
			}
		}
		return true, m.Ast(), nil
	}
	//
	// from:
	// inject(x*x/2.000, x, 1.000)
	// to:
	// (1.000*1.000/2.000)
	//
	body := AstToStr(call.Args[0])
	vars := AstToStr(call.Args[1])
	data := AstToStr(call.Args[2])

	a, err := parser.ParseExpr(strings.Replace(body, vars, data, -1))
	if err != nil {
		return false, nil, err
	}

	return true, a, nil
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
	if len(call.Args) != 1 {
		panic("this is impossible")
	}
	id.Name = "" // TODO : why it is here?
	mt, ok := isMatrix(call.Args[0])
	if !ok {
		panic("not valid transpose matrix")
	}

	// transpose
	trans := CreateMatrix(mt.Cols, mt.Rows)
	for r := 0; r < mt.Rows; r++ {
		for c := 0; c < mt.Cols; c++ {
			trans.Args[trans.Position(c, r)] = mt.Args[mt.Position(r, c)]
		}
	}
	return true, trans.Ast(), nil
}

func (s *sm) matrixDet(e goast.Expr) (changed bool, r goast.Expr, _ error) {
	call, ok := e.(*goast.CallExpr)
	if !ok {
		return false, nil, nil
	}
	id, ok := call.Fun.(*goast.Ident)
	if !ok {
		return false, nil, nil
	}
	if id.Name != det {
		return false, nil, nil
	}
	if len(call.Args) != 1 {
		panic(fmt.Errorf("this is impossible. len = %d", len(call.Args)))
	}
	id.Name = "" // TODO : why it is here?
	mt, ok := isMatrix(call.Args[0])
	if !ok {
		panic("not valid det matrix")
	}

	if mt.Cols != mt.Rows {
		panic("not square matrix")
	}

	// matrix 1x1
	if mt.Cols == 1 && mt.Rows == 1 {
		return true, mt.Args[mt.Position(0, 0)], nil
	}
	size := mt.Cols

	// determinant of matrix
	var dm goast.Expr
	dm = CreateFloat(0.0)
	for i := 0; i < size; i++ {
		mat := CreateMatrix(size-1, size-1)
		for row := 1; row < size; row++ {
			for c := 0; c < size-1; c++ {
				col := c
				if i <= col {
					col++
				}
				mat.Args[mat.Position(row-1, c)] = mt.Args[mt.Position(row, col)]
			}
		}

		determinant := &goast.CallExpr{
			Fun:  goast.NewIdent(det),
			Args: []goast.Expr{mat.Ast()},
		}

		value := mt.Args[mt.Position(0, i)]

		if ok, n := isNumber(value); ok && n == 0.0 {
			dm = &goast.BinaryExpr{
				X:  dm,
				Op: token.ADD,
				Y:  CreateFloat(0.0),
			}
			continue
		}

		if i%2 == 0 || i == 0 {
			dm = &goast.BinaryExpr{
				X:  dm,
				Op: token.ADD,
				Y: &goast.BinaryExpr{
					X:  value,
					Op: token.MUL,
					Y:  determinant,
				},
			}
		} else {
			dm = &goast.BinaryExpr{
				X:  dm,
				Op: token.SUB,
				Y: &goast.BinaryExpr{
					X:  value,
					Op: token.MUL,
					Y:  determinant,
				},
			}
		}
	}

	return true, dm, nil
}

func (s *sm) matrixInverse(e goast.Expr) (changed bool, r goast.Expr, _ error) {
	call, ok := e.(*goast.CallExpr)
	if !ok {
		return false, nil, nil
	}
	id, ok := call.Fun.(*goast.Ident)
	if !ok {
		return false, nil, nil
	}
	if id.Name != inverse {
		return false, nil, nil
	}
	if len(call.Args) != 1 {
		panic("this is impossible")
	}
	id.Name = "" // TODO : why it is here?
	mt, ok := isMatrix(call.Args[0])
	if !ok {
		panic("not valid inverse matrix")
	}

	if mt.Cols != mt.Rows {
		panic("not square matrix")
	}
	size := mt.Cols

	var value goast.Expr
	value = &goast.BinaryExpr{
		X:  CreateFloat(1.0),
		Op: token.QUO,
		Y: &goast.CallExpr{
			Fun:  goast.NewIdent(det),
			Args: []goast.Expr{call.Args[0]},
		},
	}

	copy := s.copy()
	copy.base = AstToStr(value)
	out, err := copy.run()
	s.iter += copy.iter
	if err != nil {
		return true, nil, err
	}
	value = goast.NewIdent("(" + out + ")")

	// prepare of matrix
	mat := CreateMatrix(size, size)
	for r := 0; r < size; r++ {
		for c := 0; c < size; c++ {
			part := CreateMatrix(size-1, size-1)
			for row := 0; row < size-1; row++ {
				for col := 0; col < size-1; col++ {
					row2, col2 := row, col
					if r <= row2 {
						row2++
					}
					if c <= col2 {
						col2++
					}
					part.Args[part.Position(row, col)] = mt.Args[mt.Position(row2, col2)]
				}
			}
			body := append([]goast.Expr{}, part.Args...)
			body = append(body, CreateFloat(size-1))
			body = append(body, CreateFloat(size-1))
			detm := &goast.CallExpr{
				Fun:  goast.NewIdent(det),
				Args: []goast.Expr{part.Ast()},
			}
			mat.Args[mat.Position(r, c)] = detm
			if (r+c)%2 != 0 {
				mat.Args[mat.Position(r, c)] = &goast.BinaryExpr{
					X:  CreateFloat(-1.0),
					Op: token.MUL,
					Y:  mat.Args[mat.Position(r, c)],
				}
			}
		}
	}

	result := &goast.BinaryExpr{
		X:  value,
		Op: token.MUL,
		Y: &goast.CallExpr{
			Fun:  goast.NewIdent(transpose),
			Args: []goast.Expr{mat.Ast()},
		},
	}
	return true, result, nil
}

func (s *sm) matrixSum(a goast.Expr) (changed bool, r goast.Expr, _ error) {
	// matrix(...)*matrix(...)
	bin, ok := a.(*goast.BinaryExpr)
	if !ok {
		return false, nil, nil
	}
	if !(bin.Op == token.ADD || bin.Op == token.SUB) {
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
	if left.Rows != right.Rows {
		return false, nil, fmt.Errorf("not valid matrix rows add")
	}
	if left.Cols != right.Cols {
		return false, nil, fmt.Errorf("not valid matrix columns add")
	}

	result := CreateMatrix(left.Rows, left.Cols)
	for r := 0; r < left.Rows; r++ {
		for c := 0; c < left.Cols; c++ {
			pos := left.Position(r, c)
			result.Args[pos] = &goast.BinaryExpr{
				X:  left.Args[pos], // left
				Op: bin.Op,
				Y:  right.Args[pos], // right
			}
		}
	}
	return true, result.Ast(), nil
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
	if left.Cols != right.Rows {
		return false, nil, fmt.Errorf("not valid matrix multiplication")
	}

	var result Matrix
	result.Rows = left.Rows
	result.Cols = right.Cols
	// multiplication
	for lr := 0; lr < left.Rows; lr++ {
		for rc := 0; rc < right.Cols; rc++ {
			var arg goast.Expr
			for p := 0; p < left.Cols; p++ {
				mul := &goast.BinaryExpr{
					X:  left.Args[left.Position(lr, p)],   // left
					Op: token.MUL,                         // *
					Y:  right.Args[right.Position(p, rc)], // right
				}
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
	return true, result.Ast(), nil
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
		X: &goast.BinaryExpr{
			X:  leftBin.X,
			Op: token.MUL,
			Y:  rightBin.Y,
		},
		Op: token.QUO,
		Y: &goast.BinaryExpr{
			X:  leftBin.Y,
			Op: token.MUL,
			Y:  rightBin.X,
		},
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
	if rightBin, ok := bin.Y.(*goast.BinaryExpr); ok && rightBin.Op == token.QUO {
		// from :  a/(b/c)
		// to   :  (a*c)/b
		if ok, n := isNumber(rightBin.X); ok && n == 1 {
			return true, &goast.BinaryExpr{
				X:  bin.X,
				Op: token.MUL,
				Y:  rightBin.Y,
			}, nil
		}
		if ok, n := isNumber(bin.X); ok && n == 1 {
			return true, &goast.BinaryExpr{
				X:  rightBin.Y,
				Op: token.QUO,
				Y:  rightBin.X,
			}, nil
		}
		if ok, n := isNumber(rightBin.Y); ok && n == 1 {
			return true, &goast.BinaryExpr{
				X:  bin.X,
				Op: token.QUO,
				Y:  rightBin.X,
			}, nil
		}
		return true, &goast.BinaryExpr{
			X: &goast.BinaryExpr{
				X:  bin.X,
				Op: token.MUL,
				Y:  rightBin.Y,
			},
			Op: token.QUO,
			Y:  rightBin.X,
		}, nil
	}
	leftBin, ok := bin.X.(*goast.BinaryExpr)
	if !ok {
		return false, nil, nil
	}
	if leftBin.Op != token.QUO {
		return false, nil, nil
	}

	if ok, n := isNumber(bin.Y); ok && n == 1 {
		return true, leftBin, nil
	}
	if ok, n := isNumber(leftBin.Y); ok && n == 1 {
		return true, &goast.BinaryExpr{
			X:  leftBin.X,
			Op: token.QUO,
			Y:  bin.Y,
		}, nil
	}

	//  from :  (a / b) / c
	//  to   :  a / (b * c)
	return true, &goast.BinaryExpr{
		X:  leftBin.X,
		Op: token.QUO,
		Y: &goast.BinaryExpr{
			X:  leftBin.Y,
			Op: token.MUL,
			Y:  bin.Y,
		},
	}, nil
}

func (s *sm) binaryNumber(a goast.Expr) (changed bool, r goast.Expr, _ error) {
	bin, ok := a.(*goast.BinaryExpr)
	if !ok {
		return false, nil, nil
	}

	if bin.Op == token.MUL {
		// from : (any1/any2) * (any3/any4)
		// to   : (any1 * any3) / (any2 * any4)
		if left, ok := bin.X.(*goast.BinaryExpr); ok && left.Op == token.QUO {
			if right, ok := bin.Y.(*goast.BinaryExpr); ok && right.Op == token.QUO {
				return true, &goast.BinaryExpr{
					X: &goast.BinaryExpr{
						X:  left.X,
						Op: token.MUL,
						Y:  right.X,
					},
					Op: token.QUO,
					Y: &goast.BinaryExpr{
						X:  left.Y,
						Op: token.MUL,
						Y:  right.Y,
					},
				}, nil
			}
		}

		// from : (any1/any2) * any3
		// to   : (any1 * any3) / any2
		if left, ok := bin.X.(*goast.BinaryExpr); ok && left.Op == token.QUO {
			right, ok := bin.Y.(*goast.BinaryExpr)
			if !ok || (ok && right.Op == token.QUO) {
				return true, &goast.BinaryExpr{
					X: &goast.BinaryExpr{
						X:  left.X,
						Op: token.MUL,
						Y:  bin.Y,
					},
					Op: token.QUO,
					Y:  left.Y,
				}, nil
			}
		}
	}

	// 	if q := parseQuoArray(a); 1 < len(ma) {
	// 		for i := 1; i < len(ma); i++ {
	// 			bef, bok := ma[i-1].(*goast.BinaryExpr)
	// 			pre, pok := ma[i].(*goast.BinaryExpr)
	// 			if bok && bef.Op == token.QUO {
	// 				if pok && pre.Op == token.QUO {
	// 					continue
	// 				}
	// 				// swap
	// 				ma[i-1], ma[i] = ma[i], ma[i-1]
	// 				return true, ma.toAst(), nil
	// 			}
	//
	// 			ok1, v1 := isNumber(ma[i-1])
	// 			ok2, v2 := isNumber(ma[i])
	// 			if ok1 && ok2 {
	// 				mt := ma[:i-1]
	// 				mt = append(mt, CreateFloat(v1*v2))
	// 				mt = append(mt, ma[i+1:]...)
	// 				return true, multiplySlice(mt).toAst(), nil
	// 			}
	// 		}
	// 	}

	q := parseQuoArray(a)
	if 0 < len(q.do) {
		for ui := range q.up {
			if len(q.do) == 1 {
				if ok, _ := isNumber(q.do[0]); ok {
					continue
				}
			}
			bin, ok := q.up[ui].(*goast.BinaryExpr)
			if !ok {
				continue
			}
			if !(bin.Op == token.ADD || bin.Op == token.SUB) {
				continue
			}
			if ok, _ := isNumber(bin.X); ok {
				continue
			}
			if ok, _ := isNumber(bin.Y); ok {
				continue
			}
			// from : ( ... * ( a + b )) / c
			// to   : ( ... * a ) / c + ( ... * b ) / c
			ma := parseSummArray(q.up[ui])
			if len(ma) < 2 {
				continue
			}
			common := append([]goast.Expr{}, q.up[:ui]...)
			common = append(common, q.up[ui+1:]...)
			var result goast.Expr
			for i := range ma {
				if i == 0 {
					q := quoArray{up: append(common, ma[i].toAst()), do: q.do}
					result = q.toAst()
					continue
				}
				result = &goast.BinaryExpr{
					X:  result,
					Op: token.ADD,
					Y:  quoArray{up: append(common, ma[i].toAst()), do: q.do}.toAst(),
				}
			}
			return true, result, nil
		}
	}

	if 0 < len(q.do) && 0 < len(q.up) {
		amount := 0
		upstr := make([]string, len(q.up))
		for i := range q.up {
			upstr[i] = AstToStr(q.up[i])
		}
		dostr := make([]string, len(q.do))
		for i := range q.do {
			dostr[i] = AstToStr(q.do[i])
		}
	again:
		for ui := range upstr {
			for di := range dostr {
				if upstr[ui] != dostr[di] {
					continue
				}
				q.up = append(q.up[:ui], q.up[ui+1:]...)
				q.do = append(q.do[:di], q.do[di+1:]...)
				upstr = append(upstr[:ui], upstr[ui+1:]...)
				dostr = append(dostr[:di], dostr[di+1:]...)
				amount++
				goto again
			}
		}
		if 0 < amount {
			return true, q.toAst(), nil
		}
	}

	// 	 {
	// 		if 0 < len(up) && 0 < len(do) {
	// 			var num float64 = 1.0
	// 			counter := 0
	// 		loop:
	// 			for i := range up {
	// 				if ok, n := isNumber(up[i]); ok && n != 1 {
	// 					num *= n
	// 					up = append(up[:i], up[i+1:]...)
	// 					counter++
	// 					goto loop
	// 				}
	// 			}
	// 			for i := range do {
	// 				if ok, n := isNumber(do[i]); ok && n != 1 {
	// 					num /= n
	// 					do = append(do[:i], do[i+1:]...)
	// 					counter++
	// 					goto loop
	// 				}
	// 			}
	// 			if num != 1.0 && 1 < counter {
	// 				if len(do) == 0 {
	// 					if len(up) == 0 {
	// 						return true, CreateFloat(num), nil
	// 					}
	// 					return true, &goast.BinaryExpr{
	// 						X:  CreateFloat(num),
	// 						Op: token.MUL,
	// 						Y:  up.toAst(),
	// 					}, nil
	// 				}
	// 				if len(up) == 0 {
	// 					return true, &goast.BinaryExpr{
	// 						X:  CreateFloat(num),
	// 						Op: token.QUO,
	// 						Y:  do.toAst(),
	// 					}, nil
	// 				}
	// 				return true, &goast.BinaryExpr{
	// 					X: &goast.BinaryExpr{
	// 						X:  CreateFloat(num),
	// 						Op: token.MUL,
	// 						Y:  up.toAst(),
	// 					},
	// 					Op: token.QUO,
	// 					Y:  do.toAst(),
	// 				}, nil
	// 			}
	// 		}
	// 	}

	if bin, ok := a.(*goast.BinaryExpr); ok && bin.Op == token.QUO {
		if ok, n := isNumber(bin.Y); ok && n == 1.0 {
			return true, bin.X, nil
		}
	}

	if sum := parseSummArray(a); 1 < len(sum) {
		amountNeg := 0
		for i := 0; i < len(sum); i++ {
			if i == 0 {
				continue
			}
			s := []sliceSumm(sum)
			if bin, ok := s[i].value.(*goast.BinaryExpr); ok && bin.Op == token.MUL {
				if ok, n := isNumber(bin.X); ok && n < 0 {
					s[i].isNegative = !s[i].isNegative
					if n == 0.0 {
						s[i].value = CreateFloat(0)
					} else if -n == 1.0 {
						s[i].value = bin.Y
					} else {
						s[i].value = &goast.BinaryExpr{
							X:  CreateFloat(-n),
							Op: token.MUL,
							Y:  bin.Y,
						}
					}
					amountNeg++
					i--
					continue
				}
			}
			if bin, ok := s[i].value.(*goast.BinaryExpr); ok && bin.Op == token.QUO {
				if up, ok := bin.X.(*goast.BinaryExpr); ok && up.Op == token.MUL {
					if ok, n := isNumber(up.X); ok && n < 0 {
						s[i].isNegative = !s[i].isNegative
						if n == 0 {
							s[i].value = CreateFloat(0)
						} else if -n == 1.0 {
							s[i].value = &goast.BinaryExpr{
								X:  up.Y,
								Op: token.QUO,
								Y:  bin.Y,
							}
						} else {
							s[i].value = &goast.BinaryExpr{
								X: &goast.BinaryExpr{
									X:  CreateFloat(-n),
									Op: token.MUL,
									Y:  up.Y,
								},
								Op: token.QUO,
								Y:  bin.Y,
							}
						}
						amountNeg++
						i--
						continue
					}
				}
			}
		}
		if 0 < amountNeg {
			return true, sum.toAst(), nil
		}

		type eqn struct {
			coeff float64
			ast   string
		}

		eqns := make([]eqn, len(sum))
		for i := range sum {
			eqns[i].coeff = 1.0
			if sum[i].isNegative {
				eqns[i].coeff *= -1.0
			}
			if bin, ok := sum[i].value.(*goast.BinaryExpr); ok && bin.Op == token.MUL {
				if ok, n := isNumber(bin.X); ok {
					eqns[i].coeff *= n
					eqns[i].ast = AstToStr(bin.Y)
					continue
				}
			}
			if bin, ok := sum[i].value.(*goast.BinaryExpr); ok && bin.Op == token.QUO {
				if left, ok := bin.X.(*goast.BinaryExpr); ok && left.Op == token.MUL {
					if ok, n := isNumber(left.X); ok {
						eqns[i].coeff *= n
						eqns[i].ast = AstToStr(&goast.BinaryExpr{
							X:  left.Y,
							Op: token.QUO,
							Y:  bin.Y,
						})
						continue
					}
				}
			}
			eqns[i].ast = AstToStr(sum[i].value)
		}

		size := len(eqns)
	again2:
		for i := range eqns {
			for j := range eqns {
				if i <= j {
					continue
				}
				if eqns[i].ast != eqns[j].ast {
					continue
				}
				eqns[i].coeff += eqns[j].coeff
				eqns = append(eqns[:j], eqns[j+1:]...)
				goto again2
			}
		}
		if len(eqns) < size {
			s := make([]sliceSumm, len(eqns))
			for i := range eqns {
				if eqns[i].coeff == 0 {
					s[i].value = CreateFloat(0)
					continue
				}
				if eqns[i].coeff < 0 {
					s[i].isNegative = true
					eqns[i].coeff = -eqns[i].coeff
				}
				s[i].value = &goast.BinaryExpr{
					X:  CreateFloat(eqns[i].coeff),
					Op: token.MUL,
					Y:  goast.NewIdent(eqns[i].ast),
				}
			}
			return true, summSlice(s).toAst(), nil
		}
	}

	return false, nil, nil
}

func (s *sm) oneMul(a goast.Expr) (changed bool, r goast.Expr, _ error) {
	bin, ok := a.(*goast.BinaryExpr)
	if !ok {
		return false, nil, nil
	}

	switch bin.Op {
	case token.QUO:
		// from : any / 1
		// to   : any
		ok, val := isNumber(bin.Y)
		if ok {
			if val == 1.0 {
				return true, bin.X, nil
			}
			if val == 0.0 {
				panic("cannot divide by zero")
			}
		}

	case token.MUL:
		// from : 1 * any
		// to   : any
		for _, v := range []struct {
			l, r goast.Expr
		}{
			{l: bin.X, r: bin.Y},
			{l: bin.Y, r: bin.X},
		} {
			ok, val := isNumber(v.l)
			if !ok {
				continue
			}
			if val == 1.0 {
				return true, v.r, nil
			}
			if val == 0.0 {
				return true, CreateFloat(0), nil
			}
		}
	}
	return false, nil, nil
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

	// d(matrix(...),x)
	if mt, ok := isMatrix(call.Args[0]); ok {
		for i := 0; i < len(mt.Args); i++ {
			mt.Args[i] = &goast.CallExpr{
				Fun: goast.NewIdent(differential),
				Args: []goast.Expr{
					mt.Args[i],
					call.Args[1],
				},
			}
		}
		return true, mt.Ast(), nil
	}

	// d(u + v, x) = d(u,x) + d(v,x)
	if summ := parseSummArray(call.Args[0]); 1 < len(summ) {
		var results []goast.Expr
		for i := range summ {
			results = append(results, &goast.CallExpr{
				Fun: goast.NewIdent(differential),
				Args: []goast.Expr{
					summ[i].toAst(),
					goast.NewIdent(dvar),
				}})
		}

		r, err := s.summOfParts(results)
		if err != nil {
			return false, nil, err
		}
		return true, r, err
	}

	// from : d(-...,x)
	// to   : -d(...,x)
	if un, ok := call.Args[0].(*goast.UnaryExpr); ok {
		if un.Op == token.SUB {
			return true, &goast.UnaryExpr{
				X: &goast.CallExpr{
					Fun: goast.NewIdent(differential),
					Args: []goast.Expr{
						un.X,
						call.Args[1],
					},
				},
				Op: token.SUB,
			}, nil
		} else {
			return true, &goast.CallExpr{
				Fun: goast.NewIdent(differential),
				Args: []goast.Expr{
					un.X,
					call.Args[1],
				},
			}, nil
		}
	}

	if bin, ok := call.Args[0].(*goast.BinaryExpr); ok {
		switch bin.Op {
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
				X: &goast.BinaryExpr{
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
				},
				Op: token.QUO,
				Y: &goast.BinaryExpr{
					X:  v3,
					Op: token.MUL,
					Y:  v3,
				},
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
	// from : d(-(...),x)
	// to   : -d(...,x)
	if un, ok := call.Args[0].(*goast.UnaryExpr); ok {
		return true, &goast.UnaryExpr{
			Op: un.Op,
			X: &goast.CallExpr{
				Fun: goast.NewIdent(differential),
				Args: []goast.Expr{
					un.X,
					id,
				},
			},
		}, nil
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
				if ok, _ := isNumber(exp); ok {
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
									Y:  CreateFloat("1.000"),
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
		num, _ := isNumber(call.Args[0])
		if num {
			return true, CreateFloat("0"), nil
		}
	}
	{
		// from:
		// d(x,x)
		// to:
		// 1.000
		if x, ok := call.Args[0].(*goast.Ident); ok {
			if x.Name == dvar {
				return true, CreateFloat("1"), nil
			}
		}
	}
	{
		// from:
		// d(constant,x)
		// to:
		// constant * d(1.000,x)
		if s.isConstant(call.Args[0]) {
			call.Args[0] = CreateFloat("1")
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
				return true, CreateFloat("0.0"), nil
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
		return true, CreateFloat("1"), nil
	}

	if exn == 1 {
		// from : pow(...,1)
		// to   : ...
		return true, val, nil
	}

	if exn < 0 {
		// from : pow(...,-5)
		// to   : 1/pow(...,5)
		return true, &goast.BinaryExpr{
			X:  CreateFloat(1),
			Op: token.QUO,
			Y: &goast.CallExpr{
				Fun: goast.NewIdent(pow),
				Args: []goast.Expr{
					val,
					CreateFloat(fmt.Sprintf("%d", -exn)),
				},
			},
		}, nil
	}

	if exn%2 == 0 {
		// from : pow(...,4)
		// to   : pow(...,2)*pow(...,2)
		copy := s.copy()
		copy.base = AstToStr(&goast.CallExpr{
			Fun: goast.NewIdent(pow),
			Args: []goast.Expr{
				val,
				CreateFloat(fmt.Sprintf("%d", exn/2)),
			},
		})
		out, err := copy.run()
		s.iter += copy.iter
		if err != nil {
			return false, nil, err
		}
		out = "(" + out + ")"
		return true, &goast.BinaryExpr{
			X:  goast.NewIdent(out),
			Op: token.MUL,
			Y:  goast.NewIdent(out),
		}, nil
	}

	return true, &goast.BinaryExpr{
		X:  val,
		Op: token.MUL,
		Y: &goast.CallExpr{
			Fun: goast.NewIdent(pow),
			Args: []goast.Expr{
				val,
				CreateFloat(fmt.Sprintf("%d", exn-1)),
			},
		},
	}, nil
}

func (s *sm) openParen(a goast.Expr) (changed bool, r goast.Expr, _ error) {
	ma := parseQuoArray(a)
	if len(ma.up) < 2 {
		return false, nil, nil
	}

	// from : (... -+ ... -+ ...) * (... -+ ... -+ ...)
	// to   :
	do := ma.do

	ss := make([][]sliceSumm, len(ma.up))
	ok := false
	size := 1
	for i := range ma.up {
		ss[i] = parseSummArray(ma.up[i])
		if 1 < len(ss[i]) {
			ok = true
		}
		size *= len(ss[i])
	}

	if !ok {
		return false, nil, nil
	}

	results := make([]goast.Expr, size)
	for ir := range results {
		results[ir] = CreateFloat(1)
	}
	repeat := size
	for i := range ss {
		repeat /= len(ss[i])
		index := 0
		for tu := 0; tu < size/(repeat*len(ss[i])); tu++ {
			for pos := 0; pos < len(ss[i]); pos++ {
				for t := 0; t < repeat; t++ {
					results[index] = &goast.BinaryExpr{
						X:  results[index],
						Op: token.MUL,
						Y:  ss[i][pos].toAst(),
					}
					index++
				}
			}
		}
	}
	for i := range results {
		var q quoArray
		q.up = []goast.Expr{results[i]}
		q.do = do
		results[i] = q.toAst()
	}
	r, err := s.summOfParts(results)
	if err != nil {
		return false, nil, err
	}

	return true, r, err
}

func (s *sm) summOfParts(ps []goast.Expr) (r goast.Expr, _ error) {
	parse := func(p goast.Expr) (string, error) {
		copy := s.copy()
		copy.base = AstToStr(p)
		out, err := copy.run()
		s.iter += copy.iter
		if err != nil {
			return "", err
		}
		return "(" + out + ")", err
	}

	if len(ps) == 1 {
		out, err := parse(ps[0])
		return goast.NewIdent(out), err
	}

	if 2 < len(ps) {
		middle := len(ps) / 2
		var rs [2]goast.Expr
		var errs [2]error
		var wg sync.WaitGroup
		wg.Add(2)
		for _, v := range []struct {
			index int
			ps    []goast.Expr
		}{
			{0, ps[:middle]},
			{1, ps[middle:]},
		} {
			index := v.index
			ps := v.ps
			go func(i int, ps []goast.Expr) {
				rs[i], errs[i] = s.summOfParts(ps)
				wg.Done()
			}(index, ps)
		}
		wg.Wait()
		if errs[0] != nil {
			return nil, errs[0]
		}
		if errs[1] != nil {
			return nil, errs[0]
		}
		return s.summOfParts([]goast.Expr{rs[0], rs[1]})
	}

	var result goast.Expr
	for i := range ps {
		out, err := parse(ps[i])
		if err != nil {
			return nil, err
		}

		if i == 0 {
			result = goast.NewIdent(out)
			continue
		}
		result = &goast.BinaryExpr{
			X:  result,
			Op: token.ADD,
			Y:  goast.NewIdent(out),
		}

		out, err = parse(result)
		if err != nil {
			return nil, err
		}

		result = goast.NewIdent(out)
	}
	return result, nil
}

func (s *sm) insideParen(a goast.Expr) (changed bool, r goast.Expr, _ error) {
	if u, ok := a.(*goast.ParenExpr); ok {
		return true, u.X, nil
	}
	if bin, ok := a.(*goast.BinaryExpr); ok && (bin.Op == token.ADD || bin.Op == token.SUB) {
		ok := false
		if u, ok := bin.X.(*goast.ParenExpr); ok {
			bin.X = u.X
			ok = true
		}
		if u, ok := bin.Y.(*goast.ParenExpr); ok {
			bin.Y = u.X
			ok = true
		}
		if ok {
			return true, &goast.BinaryExpr{X: bin.X, Op: bin.Op, Y: bin.Y}, nil
		}
	}
	return false, nil, nil
}

func (s *sm) zeroValueMul(e goast.Expr) (changed bool, r goast.Expr, _ error) {
	bin, ok := e.(*goast.BinaryExpr)
	if !ok {
		return false, nil, nil
	}

	isZero := func(e goast.Expr) bool {
		ok, v := isNumber(e)
		if !ok {
			return false
		}
		return v == 0.0
	}

	if bin.Op == token.ADD {
		if isZero(bin.X) {
			// zero + any
			return true, bin.Y, nil
		}
		if isZero(bin.Y) {
			// any + zero
			return true, bin.X, nil
		}
	}

	if bin.Op == token.SUB {
		// TODO
		// if isZero(bin.X) {
		// 	// zero - any
		// 	return true, bin.Y, nil
		// }
		if isZero(bin.Y) {
			// any - zero
			return true, bin.X, nil
		}
	}

	// any * zero
	// zero * any
	if bin.Op == token.MUL && (isZero(bin.X) || isZero(bin.Y)) {
		return true, CreateFloat(0.0), nil
	}

	// zero / any
	if bin.Op == token.QUO && isZero(bin.X) {
		return true, CreateFloat(0.0), nil
	}

	return false, nil, nil
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
	if len(call.Args) != 4 {
		panic("not valid integral")
	}

	var (
		function = call.Args[0]
		variable = call.Args[1]
		begin    = call.Args[2]
		finish   = call.Args[3]
	)

	if !s.isVariable(variable) {
		return false, nil, fmt.Errorf("Variable of integral is not variable: %s", AstToStr(variable))
	}

	// integral(...+...)
	// integral(...)+integral(...)
	if summ := parseSummArray(function); 1 < len(summ) {
		var results []goast.Expr
		for i := range summ {
			results = append(results, &goast.CallExpr{
				Fun: goast.NewIdent(integralName),
				Args: []goast.Expr{
					summ[i].toAst(),
					variable,
					begin,
					finish,
				}})
		}

		r, err := s.summOfParts(results)
		if err != nil {
			return false, nil, err
		}
		return true, r, err
	}

	// integral(matrix(...),x,0,1)
	if mt, ok := isMatrix(function); ok {
		for i := 0; i < len(mt.Args); i++ {
			mt.Args[i] = &goast.CallExpr{
				Fun: goast.NewIdent(integralName),
				Args: []goast.Expr{
					mt.Args[i],
					variable, begin, finish,
				},
			}
		}
		return true, mt.Ast(), nil
	}

	// extract constansts:
	// for example:
	//	integral(a       , ...)
	// to:
	//	a * integral(1.000 , ...)
	//
	//	integral(a * ... , ...)
	// to:
	//	a * integral(1.000 * ... , ...)
	//
	//	integral(a / ... , ...)
	// to:
	//	a * integral(1.000 / ... , ...)
	var possibleExtract func(e goast.Expr) (result bool)
	possibleExtract = func(e goast.Expr) (result bool) {
		//	defer func(){
		//		fmt.Println(	">>", AstToStr(e), result)
		//	}()
		// constants or numbers
		if ok, v := isNumber(e); ok && v != 1.0 {
			return true
		}
		if s.isConstant(e) {
			return true
		}
		// trigonometric of constants or numbers
		if call, ok := e.(*goast.CallExpr); ok {
			ok = false
			for _, name := range []string{sinName, cosName, tanName} {
				var id *goast.Ident
				id, ok = call.Fun.(*goast.Ident)
				if !ok {
					continue
				}
				if id.Name != name {
					return true
				}
			}
		}
		// number/number
		// constants/constants
		if bin, ok := e.(*goast.BinaryExpr); ok {
			if possibleExtract(bin.X) && possibleExtract(bin.Y) {
				return true
			}
		}

		return false
	}

	// from:
	// integral(a * ...)
	// to:
	// a*integral(...)
	{
		q := parseQuoArray(function)
		var (
			coeff      goast.Expr = goast.NewIdent("1.000")
			addedCoeff bool       = false
		)
	again:
		for i := 0; i < len(q.up); i++ {
			if !possibleExtract(q.up[i]) {
				continue
			}
			addedCoeff = true
			coeff = &goast.BinaryExpr{
				X:  coeff,
				Op: token.MUL,
				Y:  q.up[i],
			}
			q.up = append(q.up[:i], q.up[i+1:]...)
			goto again
		}
		for i := 0; i < len(q.do); i++ {
			if !possibleExtract(q.do[i]) {
				continue
			}
			addedCoeff = true
			coeff = &goast.BinaryExpr{
				X:  coeff,
				Op: token.MUL,
				Y: &goast.BinaryExpr{
					X:  CreateFloat(1),
					Op: token.QUO,
					Y:  q.do[i],
				},
			}
			q.do = append(q.do[:i], q.do[i+1:]...)
			goto again
		}
		if addedCoeff {
			if len(q.up) == 0 && len(q.do) == 0 {
				return true, &goast.BinaryExpr{
					X:  coeff,
					Op: token.MUL,
					Y: &goast.CallExpr{
						Fun: goast.NewIdent(integralName),
						Args: []goast.Expr{
							CreateFloat(1),
							variable, begin, finish,
						},
					},
				}, nil
			}
			return true, &goast.BinaryExpr{
				X:  coeff,
				Op: token.MUL,
				Y: &goast.CallExpr{
					Fun: goast.NewIdent(integralName),
					Args: []goast.Expr{
						q.toAst(),
						variable, begin, finish,
					},
				},
			}, nil
			// 			if len(q.up) == 0 {
			// 				return true, &goast.BinaryExpr{
			// 					X:  coeff,
			// 					Op: token.MUL,
			// 					Y: &goast.CallExpr{
			// 						Fun: goast.NewIdent(integralName),
			// 						Args: []goast.Expr{
			// 							&goast.BinaryExpr{
			// 								X:  CreateFloat(1),
			// 								Op: token.QUO,
			// 								Y:  q.do.toAst(),
			// 							},
			// 							variable, begin, finish,
			// 						},
			// 					},
			// 				}, nil
			// 			}
			// 			if len(q.do) == 0 {
			// 				return true, &goast.BinaryExpr{
			// 					X:  coeff,
			// 					Op: token.MUL,
			// 					Y: &goast.CallExpr{
			// 						Fun: goast.NewIdent(integralName),
			// 						Args: []goast.Expr{
			// 							q.up.toAst(),
			// 							variable, begin, finish,
			// 						},
			// 					},
			// 				}, nil
			// 			}
			// 			return true, &goast.BinaryExpr{
			// 				X:  coeff,
			// 				Op: token.MUL,
			// 				Y: &goast.CallExpr{
			// 					Fun: goast.NewIdent(integralName),
			// 					Args: []goast.Expr{
			// 						&goast.BinaryExpr{
			// 							X:  q.up.toAst(),
			// 							Op: token.QUO,
			// 							Y:  q.do.toAst(),
			// 						},
			// 						variable, begin, finish,
			// 					},
			// 				},
			// 			}, nil
		}
	}

	//
	// d(pow(x,n+1)/(n+1), 0.000, 1.000)
	//
	// n = 1
	// integral(x, x, 0.000, 1.000)
	// inject(pow(x,1+1)/(1+1), x, 1) - inject(pow(x,1+1)/(1+1), x, 0)
	//
	{
		body := AstToStr(function)
		n := float64(strings.Count(body, AstToStr(variable)))
		body = strings.Replace(body, AstToStr(variable), "", -1)
		body = strings.Replace(body, "(", "", -1)
		body = strings.Replace(body, ")", "", -1)
		body = strings.Replace(body, "*", "", -1)
		body = strings.TrimSpace(body)

		if body == "" {
			power := &goast.CallExpr{
				Fun: goast.NewIdent("pow"),
				Args: []goast.Expr{
					variable,
					CreateFloat(fmt.Sprintf("%15e", n+1.0)),
				},
			}
			div := &goast.BinaryExpr{
				X:  power,
				Op: token.QUO,
				Y:  CreateFloat(fmt.Sprintf("%15e", n+1.0)),
			}
			return true, &goast.BinaryExpr{
				X: &goast.CallExpr{
					Fun: goast.NewIdent("inject"),
					Args: []goast.Expr{
						div,
						variable,
						finish,
					},
				},
				Op: token.SUB,
				Y: &goast.CallExpr{
					Fun: goast.NewIdent("inject"),
					Args: []goast.Expr{
						div,
						variable,
						begin,
					},
				},
			}, nil
		}

		if n == 0 {
			return true, &goast.BinaryExpr{
				X:  function,
				Op: token.MUL,
				Y: &goast.BinaryExpr{
					X:  finish,
					Op: token.SUB,
					Y:  begin,
				},
			}, nil
		}
	}

	return false, nil, nil
}

func (s *sm) mulConstToMatrix(a goast.Expr) (changed bool, r goast.Expr, _ error) {
	v, ok := a.(*goast.BinaryExpr)
	if !ok {
		return false, nil, nil
	}
	if v.Op == token.QUO {
		_, ok = isMatrix(v.Y)
		if ok {
			panic("not valid matrix quo")
		}
		ok = isTranspose(v.Y)
		if ok {
			panic("not valid transpose quo")
		}
		mt, ok := isMatrix(v.X)
		if !ok {
			return false, nil, nil
		}
		for i := 0; i < len(mt.Args); i++ {
			mt.Args[i] = &goast.BinaryExpr{
				X:  mt.Args[i],
				Op: token.QUO,
				Y:  v.Y,
			}
		}
		return true, mt.Ast(), nil
	}

	if v.Op != token.MUL {
		return false, nil, nil
	}

	value, matExpr := v.X, v.Y
	for i := 0; i < 2; i++ {
		value, matExpr = matExpr, value
		if _, ok := isMatrix(matExpr); !ok {
			continue
		}
		if ok := isTranspose(value); ok {
			continue
		}
		if _, ok := isMatrix(value); ok {
			continue
		}

		mt, ok := isMatrix(matExpr)
		if !ok {
			continue
		}
		for i := 0; i < len(mt.Args); i++ {
			mt.Args[i] = &goast.BinaryExpr{
				X:  mt.Args[i],
				Op: token.MUL, // *
				Y:  value,
			}
		}
		return true, mt.Ast(), nil
	}

	return false, nil, nil
}

var sortCounter [5]int

func (s *sm) sort(a goast.Expr) (changed bool, r goast.Expr, _ error) {
	// defer func(){
	// 	fmt.Println("sort counter: ", sortCounter)
	// }()
	if summ := parseSummArray(a); 0 < len(summ) {
		{
			sort := func(es []goast.Expr) (changed bool) {
				amount := 0
				estr := make([]string, len(es))
				for i := range es {
					estr[i] = AstToStr(es[i])
				}
			again:
				runAgain := false
				for i := 1; i < len(es); i++ {
					if ok, _ := isNumber(es[i-1]); ok {
						continue
					}
					if !s.isConstant(es[i-1]) && s.isConstant(es[i]) {
						es[i-1], es[i] = es[i], es[i-1]
						estr[i-1], estr[i] = estr[i], estr[i-1]
						amount++
						runAgain = true
					}
					if s.isConstant(es[i-1]) && s.isConstant(es[i]) {
						if estr[i-1] > estr[i] {
							es[i-1], es[i] = es[i], es[i-1]
							estr[i-1], estr[i] = estr[i], estr[i-1]
							amount++
							runAgain = true
						}
					}
				}
				if runAgain {
					goto again
				}
				if 0 < amount {
					return true
				}
				return false
			}
			amount := 0
			for i := range summ {
				q := parseQuoArray(summ[i].value)
				u, d := sort(q.up), sort(q.do)
				if u || d {
					summ[i].value = q.toAst()
					amount++
				}
			}
			if 0 < amount {
				sortCounter[0]++
				return true, summ.toAst(), nil
			}
		}
		{
			amountgl := 0
			for i := range summ {
				q := parseQuoArray(summ[i].value)

				var firstNumber bool
				if 0 < len(q.up) {
					firstNumber, _ = isNumber(q.up[0])
				}
				var numbers float64 = 1
				amount := 0
				for i := 0; i < len(q.up); i++ {
					if ok, n := isNumber(q.up[i]); ok {
						numbers *= n
						q.up = append(q.up[:i], q.up[i+1:]...)
						i--
						amount++
					}
				}
				for i := 0; i < len(q.do); i++ {
					if ok, n := isNumber(q.do[i]); ok {
						numbers /= n
						q.do = append(q.do[:i], q.do[i+1:]...)
						i--
						amount++
					}
				}
				if numbers != 1.0 {
					q.up = append(q.up, CreateFloat(numbers))
					q.up[0], q.up[len(q.up)-1] = q.up[len(q.up)-1], q.up[0]
				}

				if 1 < amount {
					changed = true
				}
				if !firstNumber && 1 == amount {
					changed = true
				}
				if changed {
					sortCounter[1]++
					amountgl++
					summ[i].value = q.toAst()
				}
			}
			if 0 < amountgl {
				return true, summ.toAst(), nil
			}
		}
		{
			amount := 0
			for i := range summ {
				bin, ok := summ[i].value.(*goast.BinaryExpr)
				if !ok || bin.Op != token.QUO {
					continue
				}
				if un, ok := bin.X.(*goast.UnaryExpr); ok {
					summ[i].value = &goast.UnaryExpr{
						Op: un.Op,
						X: &goast.BinaryExpr{
							X:  un.X,
							Op: token.QUO,
							Y:  bin.Y,
						},
					}
					amount++
				}
			}
			if 0 < amount {
				sortCounter[2]++
				return true, summ.toAst(), nil
			}
		}

		{
			if 1 < len(summ) {
				amount := 0
				for i := range summ {
					if i == 0 {
						continue
					}
					if un, ok := summ[i].value.(*goast.UnaryExpr); ok {
						summ[i].value = un.X
						if un.Op == token.SUB {
							summ[i].isNegative = !summ[i].isNegative
						}
						amount++
					}
				}
				if 0 < amount {
					sortCounter[3]++
					return true, summ.toAst(), nil
				}
			}
		}
		{
			for i := 1; i < len(summ); i++ {
				if ok, _ := isNumber(summ[i].value); ok {
					summ[0], summ[i] = summ[i], summ[0] // swap
					sortCounter[4]++
					return true, summ.toAst(), nil
				}
			}
		}

	}

	return false, nil, nil
}

func (s *sm) constants(a goast.Expr) (changed bool, r goast.Expr, _ error) {
	if summ := parseSummArray(a); 1 < len(summ) {
		var numbers float64 = 0.0
		amount := 0
		for i := 0; i < len(summ); i++ {
			ok, n := isNumber(summ[i].value)
			if !ok {
				continue
			}
			if summ[i].isNegative {
				numbers -= n
			} else {
				numbers += n
			}
			if len(summ) == 1 {
				return true, CreateFloat(numbers), nil
			}
			summ = append(summ[:i], summ[i+1:]...)
			amount++
			i--
			continue
		}
		if (1 < amount) || (1 == amount && numbers == 0) {
			if numbers == 0 {
				return true, summ.toAst(), nil
			}
			return true, &goast.BinaryExpr{
				X:  CreateFloat(numbers),
				Op: token.ADD,
				Y:  summ.toAst(),
			}, nil
		}
	}

	v, ok := a.(*goast.BinaryExpr)
	if !ok {
		return false, nil, nil
	}
	// constants + constants
	xOk, x := isNumber(v.X)
	yOk, y := isNumber(v.Y)
	if !(xOk && yOk) {
		return false, nil, nil
	}

	if y == 0.0 && v.Op == token.QUO {
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

	return true, CreateFloat(fmt.Sprintf("%.15e", result)), nil
}

// FloatFormat is format of float value, for more precision calculation use
// value equal 12.
var FloatFormat int = 3

func CreateFloat(value interface{}) *goast.BasicLit {
	switch v := value.(type) {
	case float64:
		format := fmt.Sprintf("%%.%df", FloatFormat)
		return &goast.BasicLit{
			Kind:  token.FLOAT,
			Value: fmt.Sprintf(format, v),
		}
	case int:
		return CreateFloat(float64(v))
	case string:
		val, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			panic(fmt.Errorf("`%s` : %v", value, err))
		}
		return CreateFloat(val)
	}
	panic(fmt.Errorf("CreateFloat: %#v", value))
}

func isNumber(node goast.Node) (ok bool, val float64) {
	if un, ok := node.(*goast.UnaryExpr); ok {
		ok, val = isNumber(un.X)
		if un.Op == token.SUB {
			return ok, val * (-1)
		}
		return ok, val
	}
	if par, ok := node.(*goast.ParenExpr); ok {
		return isNumber(par.X)
	}
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

type Matrix struct {
	Args       []goast.Expr
	Rows, Cols int
}

func (m Matrix) Ast() goast.Expr {
	body := append([]goast.Expr{}, m.Args...)
	body = append(body, CreateFloat(m.Rows))
	body = append(body, CreateFloat(m.Cols))
	return &goast.CallExpr{
		Fun:  goast.NewIdent(matrix),
		Args: body,
	}
}

func (m Matrix) Position(r, c int) int {
	if m.Rows <= r {
		panic("matrix ouside of rows")
	}
	if m.Cols <= c {
		panic("matrix ouside of columns")
	}
	return c + m.Cols*r
}

func (m Matrix) String() string {
	var out string
	for r := 0; r < m.Rows; r++ {
		for c := 0; c < m.Cols; c++ {
			if ok, n := isNumber(m.Args[m.Position(r, c)]); ok && n == 0.0 {
				// do not print zero values
				continue
			}
			out += fmt.Sprintf("[%2d,%2d] : %s\n", r, c, AstToStr(m.Args[m.Position(r, c)]))
		}
	}
	return out
}

func CreateMatrix(r, c int) (m *Matrix) {
	m = new(Matrix)
	m.Rows = r
	m.Cols = c
	m.Args = make([]goast.Expr, r*c)
	for pos := range m.Args {
		m.Args[pos] = CreateFloat(0)
	}
	return
}

func ParseMatrix(str string) (m *Matrix, ok bool) {
	expr, err := parser.ParseExpr(str)
	if err != nil {
		return nil, false
	}
	return isMatrix(expr)
}

func isMatrix(e goast.Expr) (mt *Matrix, ok bool) {
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
	mt = new(Matrix)
	if len(call.Args) < 3 {
		panic(fmt.Errorf("matrix is not valid: %d\n%#v\n%s", len(call.Args), call, AstToStr(call)))
	}
	mt.Args = call.Args[:len(call.Args)-2]
	// parse rows and columns
	ok, v := isNumber(call.Args[len(call.Args)-2])
	if !ok {
		return nil, false
	}
	mt.Rows = int(v)

	ok, v = isNumber(call.Args[len(call.Args)-1])
	if !ok {
		return nil, false
	}
	mt.Cols = int(v)

	if len(mt.Args) != mt.Rows*mt.Cols {
		panic(fmt.Errorf("not valid matrix: args=%d rows=%d columns=%d",
			len(mt.Args),
			mt.Rows,
			mt.Cols,
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
	fmt.Println(AstToStr(e))
	goast.Print(token.NewFileSet(), e)
	fmt.Println("-------------")
}

type quoArray struct{ up, do []goast.Expr }

func (q quoArray) toAst() goast.Expr {
	var upper, downer goast.Expr
	for i := range q.up {
		if i == 0 {
			upper = q.up[i]
			continue
		}
		upper = &goast.BinaryExpr{
			X:  upper,
			Op: token.MUL,
			Y:  q.up[i],
		}
	}
	for i := range q.do {
		if i == 0 {
			downer = q.do[i]
			continue
		}
		downer = &goast.BinaryExpr{
			X:  downer,
			Op: token.MUL,
			Y:  q.do[i],
		}
	}
	if len(q.do) == 0 && len(q.up) == 0 {
		return CreateFloat(1)
	}
	if len(q.up) == 0 {
		upper = CreateFloat(1)
	}
	if len(q.do) == 0 {
		return upper
	}
	return &goast.BinaryExpr{
		X:  upper,
		Op: token.QUO,
		Y:  downer,
	}
}

func parseQuoArray(e goast.Expr) (q quoArray) {
	switch v := e.(type) {
	case *goast.ParenExpr:
		return parseQuoArray(v.X)
	case *goast.UnaryExpr:
		qe := parseQuoArray(v.X)
		if v.Op == token.SUB {
			qe.up = append(qe.up, CreateFloat(-1))
			qe.up[0], qe.up[len(qe.up)-1] = qe.up[len(qe.up)-1], qe.up[0]
		}
		return qe
	case *goast.BinaryExpr:
		switch v.Op {
		case token.MUL:
			x := parseQuoArray(v.X)
			y := parseQuoArray(v.Y)
			q.up = append(x.up, y.up...)
			q.do = append(x.do, y.do...)
			return
		case token.QUO: // x / y
			x := parseQuoArray(v.X)
			y := parseQuoArray(v.Y)
			q.up = append(x.up, y.do...)
			q.do = append(x.do, y.up...)
			return
		}
	}
	q.up = append(q.up, e)
	return
}

type summSlice []sliceSumm

func (s summSlice) toAst() goast.Expr {
	if len(s) == 0 {
		panic("not valid summSlice")
	}
	v := s[0].toAst()
	for i := 1; i < len(s); i++ {
		if s[i].isNegative {
			v = &goast.BinaryExpr{X: v, Op: token.SUB, Y: s[i].value}
		} else {
			v = &goast.BinaryExpr{X: v, Op: token.ADD, Y: s[i].value}
		}
	}
	return v
}

type sliceSumm struct {
	isNegative bool
	value      goast.Expr
}

func (s sliceSumm) toAst() goast.Expr {
	if !s.isNegative {
		return s.value
	}
	return &goast.UnaryExpr{Op: token.SUB, X: s.value}
}

func parseSummArray(e goast.Expr) (s summSlice) {
	switch v := e.(type) {
	case *goast.ParenExpr:
		return parseSummArray(v.X)

	// DO NOT ADD
	// 	case *goast.UnaryExpr:

	case *goast.BinaryExpr:
		if v.Op == token.ADD || v.Op == token.SUB {
			left, right := parseSummArray(v.X), parseSummArray(v.Y)
			if v.Op == token.SUB {
				for i := range right {
					right[i].isNegative = !right[i].isNegative
				}
			}
			return append(left, right...)
		}
	}
	return append(s, sliceSumm{isNegative: false, value: e})
}
