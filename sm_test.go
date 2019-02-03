package sm_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Konstantin8105/sm"
)

func Test(t *testing.T) {
	tcs := []struct {
		expr string
		out  string
	}{
		{
			expr: "1+2",
			out:  "3.000",
		}, {
			expr: "2*(9+3)",
			out:  "24.000",
		}, {
			expr: "(9+3)*2",
			out:  "24.000",
		}, {
			expr: "12*(2+6*6)+16/4-90/1",
			out:  "370.000",
		}, {
			expr: "a*(2+8)",
			out:  "10.000 * a",
		}, {
			expr: "a*(2+8)*a",
			out:  "10.000 * (a * a)",
		}, {
			expr: "(2+8)*a",
			out:  "10.000 * a",
		}, {
			expr: "(2+8*a)*a",
			out:  "2.000*a + 8.000*(a*a)",
		}, {
			expr: "b*(2+8*a)*a",
			out:  "2.000*(a*b) + 8.000*(a*(a*b))",
		},
		{
			expr: "b*(2+8*a)",
			out:  "2.000*b + 8.000*(a*b)",
		},
		{
			expr: "b*(2+3+8*a)",
			out:  "5.000*b + 8.000*(a*b)",
		},
		{
			expr: "b*(2+3-1+8*a)",
			out:  "4.000*b + 8.000*(a*b)",
		},
		{
			expr: "b/(2+3-1+a*8)",
			out:  "b / (4.000 + 8.000*a)",
		},
		{
			expr: "pow(4,2)",
			out:  "16.000",
		},
		{
			expr: "pow(a,2)",
			out:  "a * a",
		},
		{
			expr: "pow(a,3)",
			out:  "a * (a * a)",
		},
		{
			expr: "pow(a,-3)",
			out:  "1.000 / a * (1.000 / a * (1.000 / a))",
		},
		{
			expr: "pow(a,5-3+1)",
			out:  "a * (a * a)",
		},
		{
			expr: "pow(a+1,2)",
			out:  "1.000 + a + (a + a*a)",
		},
		{
			expr: "pow(a+b,5-4)",
			out:  "a + b",
		},
		{
			expr: "pow(a+b,4/2)",
			out:  "a*a + a*b + (a*b + b*b)",
		},
		{
			expr: "pow(2,pow(1,-1))",
			out:  "2.000",
		},
		{
			expr: "pow(1,pow(4,1))",
			out:  "1.000",
		},
		{
			expr: "-1+(-a)+(+5)+(+2+3+1)",
			out:  "10.000 - a",
		},
		{
			expr: "pow(2,pow(4,-2))",
			// TODO:
			// true value is 0.0625
			// formatting error
			out: "pow(2.000, 0.062)",
		},
		{
			expr: "pow(9,9)*4*(-3+3)*0+12.3*0-wer*0-0*wed",
			out:  "0.000",
		},

		// differential
		{
			expr: "d(pow(x,a),x);constant(a);variable(x);",
			out:  "a*pow(x,a - 1.000)",
		},
		{
			expr: "d(pow(x,2),x);variable(x);",
			out:  "2.000 * x",
		},
		{
			expr: "d(pow(a,2),x);variable(x);constant(a)",
			out:  "0.000",
		},
		{
			expr: "d(pow(a,2),x);variable(x);function(a,z)",
			out:  "0.000",
		},
		{
			expr: "d(pow(x,3),x);variable(x);",
			out:  "3.000 * (x * x)",
		},
		{
			expr: "b*d(a*x,x);constant(a);constant(b);variable(x);",
			out:  "a * b",
		},
		{
			expr: "b*d(a*x,x);constant(a);variable(x);",
			out:  "a * b",
		},
		{
			expr: "a*d(a,x);constant(a);variable(x);",
			out:  "0.000",
		},
		{
			expr: "d(2*pow(x,a),x);constant(a);variable(x);",
			out:  "2.000*(a*pow(x,a - 1.000))",
		},
		// TODO:
		// {
		// 	expr: "d(pow(x,a+1),x);constant(a);variable(x);",
		// 	out:  "(a+1)*pow(x,a)",
		// },
		{
			expr: "d(u*v,x);function(u,x);function(v,x)",
			out:  "d(u,x)*v + u*d(v,x)",
		},
		{
			expr: "d(u/v,x);function(u,x);function(v,x)",
			out:  "(d(u,x)*v - u*d(v,x)) / (v * v)",
		},
		// TODO:
		// {
		// 	expr: "d((2*(3*x-4))/(pow(x,2)+1),x);variable(x);",
		// 	out:  "2*(-3*x*x+8*x+3)/((x*x+1)*(x*x+1))",
		// },
		{
			expr: "d(u + v,x);function(u,x);function(v,x);",
			out:  "d(u,x) + d(v,x)",
		},
		{
			expr: "d(u - v,x);function(u,x);function(v,x);",
			out:  "d(u,x) - d(v,x)",
		},
		// divide by divide
		{
			expr: "(a/b)/(c/d)",
			out:  "a * d / (b * c)",
		},
		{
			expr: "a/(c/d)",
			out:  "a * d / c",
		},
	}

	for i := range tcs {
		t.Run(fmt.Sprintf("%d:%s", i, tcs[i].expr), func(t *testing.T) {
			a, err := sm.Sexpr(nil, tcs[i].expr)
			if err != nil {
				t.Fatal(err)
			}
			ac := strings.Replace(a, " ", "", -1)
			///
			ec := strings.Replace(tcs[i].out, " ", "", -1)
			if ac != ec {
				t.Fatalf("Is not same '%s' != '%s'", a, tcs[i].out)
			}
			///
			t.Log(a)
			// a2, err := sm.Sexpr(nil, tcs[i].out)
			// if err != nil {
			// 	t.Fatal(err)
			// }
			// ac2 := strings.Replace(a, " ", "", -1)
			// if ac != ac2 {
			// 	t.Fatalf("Is not same '%s' != '%s'", a, tcs[i].out)
			// }
			// t.Log(a2)
		})
	}
}
