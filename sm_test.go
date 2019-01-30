package sm_test

import (
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
			out:  "3.000000e+00",
		}, {
			expr: "2*(9+3)",
			out:  "2.400000e+01",
		}, {
			expr: "(9+3)*2",
			out:  "2.400000e+01",
		}, {
			expr: "12*(2+6*6)+16/4-90/1",
			out:  "3.700000e+02",
		},
	}

	for i := range tcs {
		t.Run(tcs[i].expr, func(t *testing.T) {
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
