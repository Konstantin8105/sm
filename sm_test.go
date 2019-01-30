package sm_test

import (
	"testing"

	"github.com/Konstantin8105/sm"
)

func Test(t *testing.T) {
	tcs := []struct {
		in, out string
	}{
		{"1+2", "3.000000e+00"},
		{"2*(9+3)", "2.400000e+01"},
		{"(9+3)*2", "2.400000e+01"},
		{"12*(2+6*6)+16/4-90/1", "3.700000e+02"},
	}

	for i := range tcs {
		t.Run(tcs[i].in, func(t *testing.T) {
			a, err := sm.Sexpr(nil, tcs[i].in)
			if err != nil {
				t.Fatal(err)
			}
			if a != tcs[i].out {
				t.Fatalf("Is not same '%s' != '%s'", a, tcs[i].out)
			}
		})
	}
}
