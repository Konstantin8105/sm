package sm

import (
	"bytes"
	"container/list"
	"fmt"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
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
					return "", fmt.Errorf("Repeat result: %s", str)
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

	changed, r, err := s.deeper(a, s.walk)
	if err != nil {
		return false, a, err
	}
	if changed {
		a, err = parser.ParseExpr(AstToStr(r))
		if err != nil {
			return
		}
		changed = true
		return changed, a, err
	}

	// try simplification
	{
		for numRule, rule := range []func(goast.Expr) (bool, goast.Expr, error){
			s.constants,
			s.openParenLeft,
			s.openParenRight,
			s.insideParen,
			s.sort,
			s.functionPow,
			s.oneMul,
			s.divide,
			s.binaryNumber,
			s.binaryUnary,
			s.zeroValueMul,
			s.differential,
			s.divideDivide,
			s.matrixTranspose,
			s.matrixDet,
			s.matrixInverse,
			s.matrixMultiply,
			s.matrixSum,
			s.mulConstToMatrix,
			s.integral,
			s.inject,
		} {
			changed, r, err := rule(a)
			if err != nil {
				return false, a, err
			}
			if changed {
				_ = numRule
				a, err = parser.ParseExpr(AstToStr(r))
				if err != nil {
					return
				}
				return true, a, err
			}
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
	value = goast.NewIdent(out)

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

func (s *sm) binaryUnary(a goast.Expr) (changed bool, r goast.Expr, _ error) {
	if u1, ok := a.(*goast.UnaryExpr); ok && u1.Op == token.SUB {
		if u2, ok := u1.X.(*goast.UnaryExpr); ok && u2.Op == token.SUB {
			return true, u2.X, nil
		}
	}

	bin, ok := a.(*goast.BinaryExpr)
	if !ok {
		return false, nil, nil
	}

	// from : (0 + ...)
	// to   : (...)
	if ok, n := isNumber(bin.X); ok && bin.Op == token.ADD && n == 0 {
		return true, bin.Y, nil
	}

	// from : (0 - ...)
	// to   : -(...)
	if ok, n := isNumber(bin.X); ok && bin.Op == token.SUB && n == 0 {
		return true, &goast.UnaryExpr{
			Op: token.SUB,
			X:  bin.Y,
		}, nil
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
	// ... * (+...)
	// to:
	// ... * (...)
	if bin.Op == token.MUL && unary.Op == token.ADD {
		return true, &goast.BinaryExpr{
			X:  bin.X,
			Op: token.MUL,
			Y:  unary.X,
		}, nil
	}

	// from:
	// ... * (-...)
	// to:
	// -(...) * ...
	if bin.Op == token.MUL && unary.Op == token.SUB {
		return true, &goast.BinaryExpr{
			X: &goast.UnaryExpr{
				Op: token.SUB,
				X:  bin.X,
			},
			Op: token.MUL,
			Y:  unary.X,
		}, nil
	}

	// from:
	// ... / (+...)
	// to:
	// ... / (...)
	if bin.Op == token.QUO && unary.Op == token.ADD {
		return true, &goast.BinaryExpr{
			X:  bin.X,
			Op: token.QUO,
			Y:  unary.X,
		}, nil
	}

	// from:
	// ... / (-...)
	// to:
	// (-...) / (...)
	if bin.Op == token.QUO && unary.Op == token.SUB {
		return true, &goast.BinaryExpr{
			X: &goast.UnaryExpr{
				Op: token.SUB,
				X:  bin.X,
			},
			Op: token.QUO,
			Y:  unary.X,
		}, nil
	}

	if bin.Op != token.ADD && bin.Op != token.SUB {
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
	if bin.Op == token.ADD && unary.Op == token.ADD {
		return true, &goast.BinaryExpr{
			X:  bin.X,
			Op: token.ADD,
			Y:  unary.X,
		}, nil
	}

	return false, nil, nil
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
	if ma, ok := parseMulArray(a); ok {
		for i := 1; i < len(ma); i++ {
			bef, bok := ma[i-1].(*goast.BinaryExpr)
			pre, pok := ma[i].(*goast.BinaryExpr)
			if bok && bef.Op == token.QUO {
				if pok && pre.Op == token.QUO {
					continue
				}
				// swap
				ma[i-1], ma[i] = ma[i], ma[i-1]
				return true, ma.toAst(), nil
			}

			ok1, v1 := isNumber(ma[i-1])
			ok2, v2 := isNumber(ma[i])
			if ok1 && ok2 {
				mt := ma[:i-1]
				mt = append(mt, CreateFloat(v1*v2))
				mt = append(mt, ma[i+1:]...)
				return true, multiplySlice(mt).toAst(), nil
			}
		}
	}

	if up, do, ok := parseQuoArray(a); ok && 0 < len(do) {
		for ui := range up {
			if bin, ok := up[ui].(*goast.BinaryExpr); ok && (bin.Op == token.ADD || bin.Op == token.SUB) {
				if len(do) == 1 {
					if ok, _ := isNumber(do[0]); ok {
						continue
					}
				}
				if ok, _ := isNumber(bin.X); ok {
					continue
				}
				if ok, _ := isNumber(bin.Y); ok {
					continue
				}
				var U multiplySlice
				U = append(U, up[:ui]...)
				U = append(U, up[ui+1:]...)
				if len(U) == 0 {
					U = append(U, CreateFloat(1.0))
				}
				return true, &goast.BinaryExpr{
					X: &goast.BinaryExpr{
						X: &goast.BinaryExpr{
							X:  U.toAst(),
							Op: token.MUL,
							Y:  bin.X,
						},
						Op: token.QUO,
						Y:  do.toAst(),
					},
					Op: bin.Op,
					Y: &goast.BinaryExpr{
						X: &goast.BinaryExpr{
							X:  U.toAst(),
							Op: token.MUL,
							Y:  bin.Y,
						},
						Op: token.QUO,
						Y:  do.toAst(),
					},
				}, nil
			}
		}
	}

	if up, do, ok := parseQuoArray(a); ok {
		amount := 0
	again:
		upstr := make([]string, len(up))
		for i := range up {
			upstr[i] = AstToStr(up[i])
		}
		dostr := make([]string, len(do))
		for i := range do {
			dostr[i] = AstToStr(do[i])
		}
		for ui := range upstr {
			for di := range dostr {
				if upstr[ui] != dostr[di] {
					continue
				}
				up = append(up[:ui], up[ui+1:]...)
				do = append(do[:di], do[di+1:]...)
				amount++
				goto again
			}
		}
		if len(up) == 0 {
			up = append([]goast.Expr{}, CreateFloat(1.0))
		}
		if len(do) == 0 {
			do = append([]goast.Expr{}, CreateFloat(1.0))
		}
		if 0 < amount {
			return true, &goast.BinaryExpr{
				X:  up.toAst(),
				Op: token.QUO,
				Y:  do.toAst(),
			}, nil
		}
	}

	if up, do, ok := parseQuoArray(a); ok {
		for i := range up {
			if un, ok := up[i].(*goast.UnaryExpr); ok {
				value := &goast.UnaryExpr{
					Op: un.Op,
					X:  CreateFloat(1),
				}
				up[i] = un.X
				up = append(up, value)
				return true, &goast.BinaryExpr{
					X:  up.toAst(),
					Op: token.QUO,
					Y:  do.toAst(),
				}, nil
			}
		}
		for i := range do {
			if un, ok := do[i].(*goast.UnaryExpr); ok {
				value := &goast.UnaryExpr{
					Op: un.Op,
					X:  CreateFloat(1),
				}
				do[i] = un.X
				do = append(do, value)
				return true, &goast.BinaryExpr{
					X:  up.toAst(),
					Op: token.QUO,
					Y:  do.toAst(),
				}, nil
			}
		}
		if 0 < len(up) && 0 < len(do) {
			var num float64 = 1.0
			counter := 0
		loop:
			for i := range up {
				if ok, n := isNumber(up[i]); ok && n != 1 {
					num *= n
					up = append(up[:i], up[i+1:]...)
					counter++
					goto loop
				}
			}
			for i := range do {
				if ok, n := isNumber(do[i]); ok && n != 1 {
					num /= n
					do = append(do[:i], do[i+1:]...)
					counter++
					goto loop
				}
			}
			if num != 1.0 && 1 < counter {
				if len(do) == 0 {
					if len(up) == 0 {
						return true, CreateFloat(num), nil
					}
					return true, &goast.BinaryExpr{
						X:  CreateFloat(num),
						Op: token.MUL,
						Y:  up.toAst(),
					}, nil
				}
				if len(up) == 0 {
					return true, &goast.BinaryExpr{
						X:  CreateFloat(num),
						Op: token.QUO,
						Y:  do.toAst(),
					}, nil
				}
				return true, &goast.BinaryExpr{
					X: &goast.BinaryExpr{
						X:  CreateFloat(num),
						Op: token.MUL,
						Y:  up.toAst(),
					},
					Op: token.QUO,
					Y:  do.toAst(),
				}, nil
			}
		}
	}

	if bin, ok := a.(*goast.BinaryExpr); ok && bin.Op == token.QUO {
		if ok, n := isNumber(bin.Y); ok && n == 1.0 {
			return true, bin.X, nil
		}
	}

	if sum := parseSummArray(a); 1 < len(sum) {
		for i := range sum {
			if i == 0 {
				continue
			}
			s := []sliceSumm(sum)
			if strings.HasPrefix(AstToStr(s[i].value), "-") {
				s[i].isNegative = !s[i].isNegative
				s[i].value = &goast.BinaryExpr{
					X:  CreateFloat(-1),
					Op: token.MUL,
					Y:  s[i].value,
				}
				return true, sum.toAst(), nil
			}
		}
		for i := range sum {
			for j := range sum {
				if j <= i {
					continue
				}
				if AstToStr(sum[i].value) != AstToStr(sum[j].value) {
					continue
				}
				if sum[i].isNegative != sum[j].isNegative {
					// remove 2 elements
					var s summSlice
					s = append(s, sum[:i]...)
					s = append(s, sum[i+1:j]...)
					s = append(s, sum[j+1:]...)
					if 0 < len(s) {
						return true, s.toAst(), nil
					} else {
						return true, CreateFloat(0), nil
					}
				}
				// summ of 2 same
				sum[i] = sliceSumm{
					isNegative: false,
					value: &goast.BinaryExpr{
						X:  CreateFloat(2),
						Op: token.MUL,
						Y:  sum[i].toAst(),
					},
				}
				sum = append(sum[:j], sum[j+1:]...)
				return true, sum.toAst(), nil
			}
		}
		for i := range sum {
			for j := range sum {
				if j <= i {
					continue
				}
				// from : a * x + b * x
				// to   : (a + b) * x
				if left, ok := parseMulArray(sum[i].toAst()); ok && 1 < len(left) {
					if right, ok := parseMulArray(sum[j].toAst()); ok && 1 < len(right) {
						if AstToStr(multiplySlice(left[1:]).toAst()) ==
							AstToStr(multiplySlice(right[1:]).toAst()) {
							ok, v1 := isNumber(left[0])
							if !ok {
								continue
							}
							ok, v2 := isNumber(right[0])
							if !ok {
								continue
							}
							sum[i] = sliceSumm{
								isNegative: false,
								value: &goast.BinaryExpr{
									X:  CreateFloat(v1 + v2),
									Op: token.MUL,
									Y:  multiplySlice(left[1:]).toAst(),
								},
							}
							sum = append(sum[:j], sum[j+1:]...)
							return true, sum.toAst(), nil
						}
					}
				}
				// from : a * x + x
				// to   : (a + 1) * x
				if left, ok := parseMulArray(sum[i].toAst()); ok && 1 < len(left) {
					valWithoutSign := sum[j].toAst()
					op := token.ADD
					if un, ok := valWithoutSign.(*goast.UnaryExpr); ok {
						valWithoutSign = un.X
						op = un.Op
					}
					if AstToStr(multiplySlice(left[1:]).toAst()) !=
						AstToStr(valWithoutSign) {
						continue
					}
					if ok, _ := isNumber(left[0]); !ok {
						continue
					}
					sum[i] = sliceSumm{
						isNegative: false,
						value: &goast.BinaryExpr{
							X: &goast.BinaryExpr{
								X:  left[0],
								Op: op,
								Y:  CreateFloat(1),
							},
							Op: token.MUL,
							Y:  multiplySlice(left[1:]).toAst(),
						},
					}
					sum = append(sum[:j], sum[j+1:]...)
					return true, sum.toAst(), nil
				}
			}
		}
	}

	leftBin, ok := bin.Y.(*goast.BinaryExpr)
	if !ok {
		return false, nil, nil
	}

	num1, num2 := bin.X, leftBin.X

	ok1, v1 := isNumber(num1)
	ok2, v2 := isNumber(num2)
	if !(ok1 && ok2) {
		return false, nil, nil
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
			X:  CreateFloat(v1 + v2),
			Op: leftBin.Op,
			Y:  leftBin.Y,
		}, nil
	}
	// from:
	// number1 - (number2 + ...)
	// to:
	// (number1 - number2) - (...)
	if bin.Op == token.SUB && leftBin.Op == token.ADD {
		return true, &goast.BinaryExpr{
			X:  CreateFloat(v1 - v2),
			Op: token.SUB,
			Y:  leftBin.Y,
		}, nil
	}
	// from:
	// number1 - (number2 - ...)
	// to:
	// (number1 - number2) + (...)
	return true, &goast.BinaryExpr{
		X:  CreateFloat(v1 - v2),
		Op: token.ADD,
		Y:  leftBin.Y,
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

	if exn > 0 {
		// from:
		// pow(..., 33)
		// to:
		// (...) * pow(..., 32)
		x1 := val
		x2 := val
		return true, &goast.BinaryExpr{
			X:  x1,
			Op: token.MUL,
			Y: &goast.CallExpr{
				Fun: goast.NewIdent(pow),
				Args: []goast.Expr{
					x2,
					CreateFloat(fmt.Sprintf("%d", exn-1)),
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
				CreateFloat(fmt.Sprintf("%d", exn+1)),
			},
		},
		Op: token.MUL,
		Y: &goast.BinaryExpr{
			X:  CreateFloat("1"),
			Op: token.QUO,
			Y:  x2,
		},
	}, nil
}

func (s *sm) openParenRight(a goast.Expr) (changed bool, r goast.Expr, _ error) {
	bin, ok := a.(*goast.BinaryExpr)
	if !ok || bin.Op != token.MUL {
		return false, nil, nil
	}

	{
		left := parseSummArray(bin.X)
		right := parseSummArray(bin.Y)
		if 1 < len(left) && 1 < len(right) {
			var results []goast.Expr
			for i := range left {
				for j := range right {
					results = append(results, &goast.BinaryExpr{
						X:  left[i].toAst(),
						Op: token.MUL,
						Y:  right[j].toAst(),
					})
				}
			}
			r, err := s.summOfParts(results)
			if err != nil {
				return false, nil, err
			}
			return true, r, err
		}
	}

	// from:
	// any * (... -+ ... -+ ...)
	// to:
	// any * ... -+ any * ... -+ any * ...
	for _, v := range []struct {
		l, r goast.Expr
	}{
		{bin.X, bin.Y},
		{bin.Y, bin.X},
	} {
		summ := parseSummArray(v.r)
		if len(summ) < 2 {
			continue
		}

		var results []goast.Expr
		for i := range summ {
			results = append(results, &goast.BinaryExpr{
				X:  v.l,
				Op: token.MUL,
				Y:  summ[i].toAst(),
			})
		}

		r, err := s.summOfParts(results)
		if err != nil {
			return false, nil, err
		}
		return true, r, err
	}

	return false, nil, nil
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
		return out, err
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

	// TODO: parallel

	var result goast.Expr
	for i := range ps {
		out, err := parse(ps[i])
		if err != nil {
			return nil, err
		}

		if i == 0 {
			result = goast.NewIdent(out)
		} else {
			result = &goast.BinaryExpr{
				X:  result,
				Op: token.ADD,
				Y:  goast.NewIdent(out),
			}
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
	return false, nil, nil
}

func (s *sm) openParenLeft(a goast.Expr) (changed bool, r goast.Expr, _ error) {
	v, ok := a.(*goast.BinaryExpr)
	if !ok {
		return false, nil, nil
	}

	if v.Op == token.ADD {
		// from:
		//	any1 + any1
		// to:
		//	2.000 * any1
		if AstToStr(v.X) == AstToStr(v.Y) {
			return true, &goast.BinaryExpr{
				X:  CreateFloat(2.000),
				Op: token.MUL,
				Y:  v.X,
			}, nil
		}

		// from:
		//	any1 + number * any1
		// to:
		//	(number+1) * any1
		x, y := v.X, v.Y
		for i := 0; i < 2; i++ {
			if right, ok := y.(*goast.BinaryExpr); ok && right.Op == token.MUL {
				if ok, _ := isNumber(right.X); ok {
					if AstToStr(x) == AstToStr(right.Y) {
						return true, &goast.BinaryExpr{
							X: &goast.BinaryExpr{
								X:  CreateFloat(1.000),
								Op: token.ADD,
								Y:  right.X,
							},
							Op: token.MUL,
							Y:  x,
						}, nil
					}
				}
			}
			x, y = y, x // swap
		}

		// from:
		//	number1 * any1 + number2 * any1
		// to:
		//	(number1+number2) * any1
		if left, ok := v.X.(*goast.BinaryExpr); ok && left.Op == token.MUL {
			if ok, _ := isNumber(left.X); ok {
				if right, ok := v.Y.(*goast.BinaryExpr); ok && right.Op == token.MUL {
					if ok, _ := isNumber(right.X); ok {
						if AstToStr(left.Y) == AstToStr(right.Y) {
							return true, &goast.BinaryExpr{
								X: &goast.BinaryExpr{
									X:  left.X,
									Op: token.ADD,
									Y:  right.X,
								},
								Op: token.MUL,
								Y:  left.Y,
							}, nil
						}
					}
				}
			}
		}
	}

	if v.Op == token.SUB {
		// from:
		//	any1 - any1
		// to:
		//	0
		if AstToStr(v.X) == AstToStr(v.Y) {
			return true, CreateFloat(0.000), nil
		}

		// from:
		//	any1 - (-number)*any2
		// to:
		//	any1 + number * any2
		if right, ok := v.Y.(*goast.BinaryExpr); ok && right.Op == token.MUL {
			if ok, val := isNumber(right.X); ok && val < 0.0 {
				return true, &goast.BinaryExpr{
					X:  v.X,
					Op: token.ADD,
					Y: &goast.BinaryExpr{
						X:  CreateFloat(-val),
						Op: token.MUL,
						Y:  right.Y,
					},
				}, nil
			}
		}
	}

	// from:
	// (... +/- ...) / any
	// to:
	// (.../any) +/- (.../any)
	if v.Op == token.QUO {
		if bin, ok := v.X.(*goast.BinaryExpr); ok && (bin.Op == token.ADD || bin.Op == token.SUB) {
			if ok, _ := isNumber(v.Y); ok || s.isConstant(v.Y) {
				return true, &goast.BinaryExpr{
					X: &goast.BinaryExpr{
						X:  bin.X,
						Op: token.QUO,
						Y:  v.Y,
					},
					Op: bin.Op,
					Y: &goast.BinaryExpr{
						X:  bin.Y,
						Op: token.QUO,
						Y:  v.Y,
					},
				}, nil
			}
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

	// integral(...+...)
	// integral(...)+integral(...)
	if summ := parseSummArray(function); 1 < len(summ) {
		prepare := func(s string) string {
			return strings.Replace(s, " ", "", -1)
		}
		start := prepare(AstToStr(function))
		var result goast.Expr
		for i := range summ {
			copy := s.copy()
			copy.base = AstToStr(&goast.CallExpr{
				Fun: goast.NewIdent(integralName),
				Args: []goast.Expr{
					summ[i].value,
					variable,
					begin,
					finish,
				}})
			out, err := copy.run()
			s.iter += copy.iter
			if err != nil {
				return true, nil, err
			}
			if summ[i].isNegative {
				out = "(-(" + out + "))"
			}
			if i == 0 {
				result = goast.NewIdent(out)
			} else {
				result = &goast.BinaryExpr{
					X:  result,
					Op: token.ADD,
					Y:  goast.NewIdent(out),
				}
			}
		}
		finish := prepare(AstToStr(result))
		if start != finish {
			return true, result, nil
		}
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
	if up, do, ok := parseQuoArray(function); ok {
		var (
			coeff      goast.Expr = goast.NewIdent("1.000")
			addedCoeff bool       = false
		)

		for i := 0; i < len(up); i++ {
			if !possibleExtract(up[i]) {
				continue
			}
			addedCoeff = true
			coeff = &goast.BinaryExpr{
				X:  coeff,
				Op: token.MUL,
				Y:  up[i],
			}
			up = append(up[:i], up[i+1:]...)
			i--
		}
		for i := 0; i < len(do); i++ {
			if !possibleExtract(do[i]) {
				continue
			}
			addedCoeff = true
			coeff = &goast.BinaryExpr{
				X:  coeff,
				Op: token.MUL,
				Y: &goast.BinaryExpr{
					X:  CreateFloat(1),
					Op: token.QUO,
					Y:  do[i],
				},
			}
			do = append(do[:i], do[i+1:]...)
			i--
		}
		if addedCoeff {
			if len(up) == 0 && len(do) == 0 {
				return true, coeff, nil
			}
			if len(up) == 0 {
				return true, &goast.BinaryExpr{
					X:  coeff,
					Op: token.MUL,
					Y: &goast.CallExpr{
						Fun: goast.NewIdent(integralName),
						Args: []goast.Expr{
							&goast.BinaryExpr{
								X:  CreateFloat(1),
								Op: token.QUO,
								Y:  do.toAst(),
							},
							variable, begin, finish,
						},
					},
				}, nil
			}
			if len(do) == 0 {
				return true, &goast.BinaryExpr{
					X:  coeff,
					Op: token.MUL,
					Y: &goast.CallExpr{
						Fun: goast.NewIdent(integralName),
						Args: []goast.Expr{
							up.toAst(),
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
						&goast.BinaryExpr{
							X:  up.toAst(),
							Op: token.QUO,
							Y:  do.toAst(),
						},
						variable, begin, finish,
					},
				},
			}, nil
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

func (s *sm) sort(a goast.Expr) (changed bool, r goast.Expr, _ error) {
	if ma, ok := parseMulArray(a); ok && 1 < len(ma) {
		firstNumber, _ := isNumber(ma[0])
		var numbers float64 = 1
		amount := 0
		for i := 0; i < len(ma); i++ {
			if ok, n := isNumber(ma[i]); ok {
				numbers *= n
				if len(ma) == 1 && i == 0 {
					return true, CreateFloat(numbers), nil
				}
				ma = append(ma[:i], ma[i+1:]...)
				i--
				amount++
			}
		}
		var result goast.Expr = ma.toAst()
		if numbers != 1.0 {
			result = &goast.BinaryExpr{
				X:  CreateFloat(numbers),
				Op: token.MUL,
				Y:  result,
			}
		}

		if 2 < amount {
			changed = true
		}
		if !firstNumber && 1 == amount {
			changed = true
		}
		if changed {
			return true, result, nil
		}
	}

	if ma, ok := parseMulArray(a); ok && 1 < len(ma) {
		amount := 0
	again:
		for i := 1; i < len(ma); i++ {
			if ok, _ := isNumber(ma[i-1]); ok {
				continue
			}
			if !s.isConstant(ma[i-1]) && s.isConstant(ma[i]) {
				ma[i-1], ma[i] = ma[i], ma[i-1]
				amount++
				goto again
			}
			if s.isConstant(ma[i-1]) && s.isConstant(ma[i]) {
				if AstToStr(ma[i-1]) > AstToStr(ma[i]) {
					ma[i-1], ma[i] = ma[i], ma[i-1]
					amount++
					goto again
				}
			}
		}
		if 0 < amount {
			return true, ma.toAst(), nil
		}
	}

	if summ := parseSummArray(a); 1 < len(summ) {
		for i := 1; i < len(summ); i++ {
			if ok, _ := isNumber(summ[i].value); ok {
				summ[0], summ[i] = summ[i], summ[0] // swap
				return true, summ.toAst(), nil
			}
		}
	}
	return false, nil, nil
}

func (s *sm) constants(a goast.Expr) (changed bool, r goast.Expr, _ error) {
	if summ := parseSummArray(a); 1 < len(summ) {
		var numbers float64 = 0.0
		amount := 0
	again:
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
			goto again
		}
		if 1 < amount {
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
	unary := 1.0
	if un, ok := node.(*goast.UnaryExpr); ok {
		if un.Op == token.SUB {
			unary = -1.0
			node = un.X
		} else if un.Op == token.ADD {
			unary = 1.0
			node = un.X
		} else {
			return false, 0.0
		}
	}
	if x, ok := node.(*goast.BasicLit); ok {
		if x.Kind == token.INT || x.Kind == token.FLOAT {
			val, err := strconv.ParseFloat(x.Value, 64)
			if err == nil {
				return true, unary * val
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

type multiplySlice []goast.Expr

func parseMulArray(e goast.Expr) (ma multiplySlice, ok bool) {
	defer func() {
		if !ok || len(ma) < 2 {
			return
		}
		for i := 0; i < len(ma); i++ {
			ok, n := isNumber(ma[i])
			if !ok {
				continue
			}
			// from : 0 * a * b * ...
			// to   : 0
			if n == 0.0 {
				ma = []goast.Expr{CreateFloat(0.0)}
				break
			}
			// from : a * 1 * b
			// to   : a * b
			if n == 1.0 {
				ma = append(ma[:i], ma[i+1:]...)
				i--
			}
		}
	}()
	if par, ok := e.(*goast.ParenExpr); ok {
		return parseMulArray(par.X)
	}
	if un, ok := e.(*goast.UnaryExpr); ok {
		mm, ok := parseMulArray(un.X)
		if !ok {
			return nil, false
		}
		if un.Op == token.SUB {
			if ok, n := isNumber(mm[0]); ok {
				mm[0] = CreateFloat(-n)
			} else {
				mm = append(mm, CreateFloat(-1))
			}
		}
		return mm, ok
	}
	if bin, ok := e.(*goast.BinaryExpr); ok && bin.Op == token.MUL {
		left, leftok := parseMulArray(bin.X)
		righ, righok := parseMulArray(bin.Y)
		if !leftok || !righok {
			return nil, false
		}
		ma = append(ma, left...)
		ma = append(ma, righ...)
		return ma, true
	}
	if bin, ok := e.(*goast.BinaryExpr); ok && bin.Op == token.QUO {
		left, leftok := parseMulArray(bin.X)
		if !leftok {
			return nil, false
		}
		ma = append(ma, left...)
		ma = append(ma, &goast.BinaryExpr{
			X:  CreateFloat(1.0),
			Op: token.QUO,
			Y:  bin.Y,
		})
		return ma, true
	}
	ma = append(ma, e)
	return ma, true
}

func (m multiplySlice) toAst() goast.Expr {
	if len(m) == 0 {
		panic("empty multiplySlice")
	}
	v := m[0]
	for i := 1; i < len(m); i++ {
		if bin, ok := m[i].(*goast.BinaryExpr); ok && bin.Op == token.QUO {
			v = &goast.BinaryExpr{
				X: &goast.BinaryExpr{
					X:  v,
					Op: token.MUL,
					Y:  bin.X,
				},
				Op: token.QUO,
				Y:  bin.Y,
			}
			continue
		}
		v = &goast.BinaryExpr{
			X:  v,
			Op: token.MUL,
			Y:  m[i],
		}
	}
	return v
}

func parseQuoArray(e goast.Expr) (up, do multiplySlice, ok bool) {
	if par, ok := e.(*goast.ParenExpr); ok {
		return parseQuoArray(par.X)
	}
	if un, ok := e.(*goast.UnaryExpr); ok {
		up, do, ok = parseQuoArray(un.X)
		if !ok {
			return nil, nil, false
		}
		if un.Op == token.SUB {
			up = append(up, CreateFloat(-1))
		}
		return up, do, ok
	}
	bin, ok := e.(*goast.BinaryExpr)
	if !ok {
		up = append(up, e)
		do = append(do, CreateFloat(1))
		ok = true
		return
	}
	switch bin.Op {
	case token.MUL:
		up2, do2, ok2 := parseQuoArray(bin.X)
		up3, do3, ok3 := parseQuoArray(bin.Y)
		if !ok2 || !ok3 {
			up = append(up, e)
			do = append(do, CreateFloat(1))
			ok = true
			return
		}
		up = append(up2, up3...)
		do = append(do2, do3...)
		if len(do) == 0 {
			do = append(do, CreateFloat(1))
		}
		ok = true

	case token.QUO:
		if up, ok = parseMulArray(bin.X); !ok {
			return
		}
		do, ok = parseMulArray(bin.Y)

	default:
		up = append(up, e)
		do = append(do, CreateFloat(1))
		ok = true
	}
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
			v = &goast.BinaryExpr{
				X:  v,
				Op: token.SUB,
				Y:  s[i].value,
			}
		} else {
			v = &goast.BinaryExpr{
				X:  v,
				Op: token.ADD,
				Y:  s[i].value,
			}
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
	return &goast.UnaryExpr{
		Op: token.SUB,
		X:  s.value,
	}
}

func parseSummArray(e goast.Expr) (s summSlice) {
	if par, ok := e.(*goast.ParenExpr); ok {
		return parseSummArray(par.X)
	}
	if un, ok := e.(*goast.UnaryExpr); ok {
		ss := parseSummArray(un.X)
		if un.Op == token.SUB {
			sd := []sliceSumm(ss)
			for i := range sd {
				sd[i].isNegative = !sd[i].isNegative
			}
		}
		return ss
	}

	bin, ok := e.(*goast.BinaryExpr)
	if !ok || !(bin.Op == token.ADD || bin.Op == token.SUB) {
		s = append(s, sliceSumm{
			isNegative: false,
			value:      e,
		})
		return
	}
	left := parseSummArray(bin.X)
	right := parseSummArray(bin.Y)
	if bin.Op == token.SUB {
		for i := range right {
			right[i].isNegative = !right[i].isNegative
		}
	}
	return append(left, right...)
}
