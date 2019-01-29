# sm
symbolic math on Go

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
```

prototype:
```golang
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
)

func main() {
	fmt.Println("Hello, playground")
	a, err := parser.ParseExpr("d*6*integer(r*8+ff,12,3)+d(f(mod(u,2)),u)")
	fmt.Println(err)

	ast.Print(token.NewFileSet(), a)
}
```

Out:
```
Hello, playground
<nil>
     0  *ast.BinaryExpr {
     1  .  X: *ast.BinaryExpr {
     2  .  .  X: *ast.BinaryExpr {
     3  .  .  .  X: *ast.Ident {
     4  .  .  .  .  NamePos: -
     5  .  .  .  .  Name: "d"
     6  .  .  .  .  Obj: *ast.Object {
     7  .  .  .  .  .  Kind: bad
     8  .  .  .  .  .  Name: ""
     9  .  .  .  .  }
    10  .  .  .  }
    11  .  .  .  OpPos: -
    12  .  .  .  Op: *
    13  .  .  .  Y: *ast.BasicLit {
    14  .  .  .  .  ValuePos: -
    15  .  .  .  .  Kind: INT
    16  .  .  .  .  Value: "6"
    17  .  .  .  }
    18  .  .  }
    19  .  .  OpPos: -
    20  .  .  Op: *
    21  .  .  Y: *ast.CallExpr {
    22  .  .  .  Fun: *ast.Ident {
    23  .  .  .  .  NamePos: -
    24  .  .  .  .  Name: "integer"
    25  .  .  .  .  Obj: *(obj @ 6)
    26  .  .  .  }
    27  .  .  .  Lparen: -
    28  .  .  .  Args: []ast.Expr (len = 3) {
    29  .  .  .  .  0: *ast.BinaryExpr {
    30  .  .  .  .  .  X: *ast.BinaryExpr {
    31  .  .  .  .  .  .  X: *ast.Ident {
    32  .  .  .  .  .  .  .  NamePos: -
    33  .  .  .  .  .  .  .  Name: "r"
    34  .  .  .  .  .  .  .  Obj: *(obj @ 6)
    35  .  .  .  .  .  .  }
    36  .  .  .  .  .  .  OpPos: -
    37  .  .  .  .  .  .  Op: *
    38  .  .  .  .  .  .  Y: *ast.BasicLit {
    39  .  .  .  .  .  .  .  ValuePos: -
    40  .  .  .  .  .  .  .  Kind: INT
    41  .  .  .  .  .  .  .  Value: "8"
    42  .  .  .  .  .  .  }
    43  .  .  .  .  .  }
    44  .  .  .  .  .  OpPos: -
    45  .  .  .  .  .  Op: +
    46  .  .  .  .  .  Y: *ast.Ident {
    47  .  .  .  .  .  .  NamePos: -
    48  .  .  .  .  .  .  Name: "ff"
    49  .  .  .  .  .  .  Obj: *(obj @ 6)
    50  .  .  .  .  .  }
    51  .  .  .  .  }
    52  .  .  .  .  1: *ast.BasicLit {
    53  .  .  .  .  .  ValuePos: -
    54  .  .  .  .  .  Kind: INT
    55  .  .  .  .  .  Value: "12"
    56  .  .  .  .  }
    57  .  .  .  .  2: *ast.BasicLit {
    58  .  .  .  .  .  ValuePos: -
    59  .  .  .  .  .  Kind: INT
    60  .  .  .  .  .  Value: "3"
    61  .  .  .  .  }
    62  .  .  .  }
    63  .  .  .  Ellipsis: -
    64  .  .  .  Rparen: -
    65  .  .  }
    66  .  }
    67  .  OpPos: -
    68  .  Op: +
    69  .  Y: *ast.CallExpr {
    70  .  .  Fun: *ast.Ident {
    71  .  .  .  NamePos: -
    72  .  .  .  Name: "d"
    73  .  .  .  Obj: *(obj @ 6)
    74  .  .  }
    75  .  .  Lparen: -
    76  .  .  Args: []ast.Expr (len = 2) {
    77  .  .  .  0: *ast.CallExpr {
    78  .  .  .  .  Fun: *ast.Ident {
    79  .  .  .  .  .  NamePos: -
    80  .  .  .  .  .  Name: "f"
    81  .  .  .  .  .  Obj: *(obj @ 6)
    82  .  .  .  .  }
    83  .  .  .  .  Lparen: -
    84  .  .  .  .  Args: []ast.Expr (len = 1) {
    85  .  .  .  .  .  0: *ast.CallExpr {
    86  .  .  .  .  .  .  Fun: *ast.Ident {
    87  .  .  .  .  .  .  .  NamePos: -
    88  .  .  .  .  .  .  .  Name: "mod"
    89  .  .  .  .  .  .  .  Obj: *(obj @ 6)
    90  .  .  .  .  .  .  }
    91  .  .  .  .  .  .  Lparen: -
    92  .  .  .  .  .  .  Args: []ast.Expr (len = 2) {
    93  .  .  .  .  .  .  .  0: *ast.Ident {
    94  .  .  .  .  .  .  .  .  NamePos: -
    95  .  .  .  .  .  .  .  .  Name: "u"
    96  .  .  .  .  .  .  .  .  Obj: *(obj @ 6)
    97  .  .  .  .  .  .  .  }
    98  .  .  .  .  .  .  .  1: *ast.BasicLit {
    99  .  .  .  .  .  .  .  .  ValuePos: -
   100  .  .  .  .  .  .  .  .  Kind: INT
   101  .  .  .  .  .  .  .  .  Value: "2"
   102  .  .  .  .  .  .  .  }
   103  .  .  .  .  .  .  }
   104  .  .  .  .  .  .  Ellipsis: -
   105  .  .  .  .  .  .  Rparen: -
   106  .  .  .  .  .  }
   107  .  .  .  .  }
   108  .  .  .  .  Ellipsis: -
   109  .  .  .  .  Rparen: -
   110  .  .  .  }
   111  .  .  .  1: *ast.Ident {
   112  .  .  .  .  NamePos: -
   113  .  .  .  .  Name: "u"
   114  .  .  .  .  Obj: *(obj @ 6)
   115  .  .  .  }
   116  .  .  }
   117  .  .  Ellipsis: -
   118  .  .  Rparen: -
   119  .  }
   120  }
```
