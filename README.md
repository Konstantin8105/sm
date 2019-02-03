# sm
symbolic math on Go

```golang
package s

func f() {
	_ = func() {
		////////////////////////////
		// code of script         //
		constant(u)
		A := [2][2]float64{{12, 22}, {33, 44}}
		constant(a)
		s := pow(u, 2) + a
		Q := A * s
		output(Q)
		////////////////////////////
	}
}
```


```go
  // Prototype example:
  parse(
    "E*F/L*integral(y*3.14,y,0,l)", // main expression
    "const(l)",                     // l is constant
    "variable(y)",                  // y is variable
  )
  
  // example 2:
  parse("12*(4+6)")                 // result will be `12*4+12*6`
  
  // example 3:
  parse("y*(12+y-2)","variable(y)") // result will be `y*10-y*y`

  // prototype of rules:
  rule("function + constant", swap)
  rule("constant * constant", sum)
  rule("mod(any,constant)", modExplode)
```



prototype:
```golang
package main

import (
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
)

func main() {
	a, _ := parser.ParseExpr("d*6*integer(r*8+ff,12,3)")
	ast.Print(token.NewFileSet(), a)
	printer.Fprint(os.Stdout, token.NewFileSet(), a)
}
```

Out:
```
0  *ast.BinaryExpr {
     1  .  X: *ast.BinaryExpr {
     2  .  .  X: *ast.Ident {
     3  .  .  .  NamePos: -
     4  .  .  .  Name: "d"
     5  .  .  .  Obj: *ast.Object {
     6  .  .  .  .  Kind: bad
     7  .  .  .  .  Name: ""
     8  .  .  .  }
     9  .  .  }
    10  .  .  OpPos: -
    11  .  .  Op: *
    12  .  .  Y: *ast.BasicLit {
    13  .  .  .  ValuePos: -
    14  .  .  .  Kind: INT
    15  .  .  .  Value: "6"
    16  .  .  }
    17  .  }
    18  .  OpPos: -
    19  .  Op: *
    20  .  Y: *ast.CallExpr {
    21  .  .  Fun: *ast.Ident {
    22  .  .  .  NamePos: -
    23  .  .  .  Name: "integer"
    24  .  .  .  Obj: *(obj @ 5)
    25  .  .  }
    26  .  .  Lparen: -
    27  .  .  Args: []ast.Expr (len = 3) {
    28  .  .  .  0: *ast.BinaryExpr {
    29  .  .  .  .  X: *ast.BinaryExpr {
    30  .  .  .  .  .  X: *ast.Ident {
    31  .  .  .  .  .  .  NamePos: -
    32  .  .  .  .  .  .  Name: "r"
    33  .  .  .  .  .  .  Obj: *(obj @ 5)
    34  .  .  .  .  .  }
    35  .  .  .  .  .  OpPos: -
    36  .  .  .  .  .  Op: *
    37  .  .  .  .  .  Y: *ast.BasicLit {
    38  .  .  .  .  .  .  ValuePos: -
    39  .  .  .  .  .  .  Kind: INT
    40  .  .  .  .  .  .  Value: "8"
    41  .  .  .  .  .  }
    42  .  .  .  .  }
    43  .  .  .  .  OpPos: -
    44  .  .  .  .  Op: +
    45  .  .  .  .  Y: *ast.Ident {
    46  .  .  .  .  .  NamePos: -
    47  .  .  .  .  .  Name: "ff"
    48  .  .  .  .  .  Obj: *(obj @ 5)
    49  .  .  .  .  }
    50  .  .  .  }
    51  .  .  .  1: *ast.BasicLit {
    52  .  .  .  .  ValuePos: -
    53  .  .  .  .  Kind: INT
    54  .  .  .  .  Value: "12"
    55  .  .  .  }
    56  .  .  .  2: *ast.BasicLit {
    57  .  .  .  .  ValuePos: -
    58  .  .  .  .  Kind: INT
    59  .  .  .  .  Value: "3"
    60  .  .  .  }
    61  .  .  }
    62  .  .  Ellipsis: -
    63  .  .  Rparen: -
    64  .  }
    65  }
d * 6 * integer(r*8+ff, 12, 3)
```
