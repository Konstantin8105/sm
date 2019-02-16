# sm
symbolic math on Go

Documentation:
```golang
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
```

Example:
```golang
func Example() {
	eq, err := sm.Sexpr(os.Stdout, "-1+(-a)+(+5)+(+2+3+1)")
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(os.Stdout, "%s", eq)
	// Output:
	// 5 + (-1 + (-a)) + (+2 + 3 + 1)
	// 5.000 + (-1 + (-a)) + (+2 + 3 + 1)
	// 5.000 + (-1 - a) + (+2 + 3 + 1)
	// 5.000 + (-1.000 - a) + (+2 + 3 + 1)
	// (5.000 + -1.000) - (a) + (+2 + 3 + 1)
	// 4.000 - (a) + (+2 + 3 + 1)
	// 4.000 - a + (+2 + 3 + 1)
	// 4.000 - a + (1 + (+2 + 3))
	// 4.000 - a + (1.000 + (+2 + 3))
	// 4.000 - a + (1.000 + 5.000)
	// 4.000 - a + 6.000
	// (6.000 + 4.000) - (a)
	// 10.000 - (a)
	// 10.000 - a
	// 10.000 - a
	// 10.000 - a
}
```
