package sm_test

import (
	"fmt"
	"testing"

	"github.com/Konstantin8105/sm"
)

func Test(t *testing.T) {
	tcs := []struct {
		expr      string
		variables []string
		out       string
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
	}

	for i := range tcs {
		t.Run(fmt.Sprintf("%d:%s", i, tcs[i].expr), func(t *testing.T) {
			a, err := sm.Sexpr(nil, tcs[i].expr, tcs[i].variables...)
			if err != nil {
				t.Fatal(err)
			}
			if a != tcs[i].out {
				t.Fatalf("Is not same '%s' != '%s'", a, tcs[i].out)
			}
		})
	}
}
