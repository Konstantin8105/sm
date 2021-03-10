package sm_test

import (
	"fmt"
	"os"
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
			expr: "2*((((9.01)+3)))",
			out:  "24.020",
		}, {
			expr: "2*((((9)+3)))",
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
			out:  "10.000 * a * a",
		}, {
			expr: "((a))*(((2+8)))*(a)",
			out:  "10.000 * a * a",
		}, {
			expr: "(2+8)*a",
			out:  "10.000 * a",
		}, {
			expr: "(2+8*a)*a",
			out:  "2.000*a + 8.000*(a*a)",
		}, {
			expr: "b*(2+8*a)*a; constant(a); constant(b)",
			out:  "2.000*(a*b) + 8.000*(a*(a*b))",
		},
		{
			expr: "b*(2+8*a); constant(a); constant(b)",
			out:  "2.000*b + 8.000*(a*b)",
		},
		{
			expr: "b*(2+3+8*a); constant(a); constant(b)",
			out:  "5.000*b + 8.000*(a*b)",
		},
		{
			expr: "(a + b) * (c - d); constant(a,b,c,d)",
			out:  "a*c - a*d + (b*c - b*d)",
		},
		{
			expr: "(a + b) * (c - d - s); constant(a,b,c,d,s)",
			out:  "a*c - a*d - a*s + (b*c - b*d - b*s)",
		},
		{
			expr: "b*(2+3-1+8*a); constant(a,b)",
			out:  "4.000*b + 8.000*(a*b)",
		},
		{
			expr: "b/(2+3-1+a*8); constant(a,b)",
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
			out:  "1.000/a*(1.000/a)*(1.000/a)",
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
			expr: "pow(a+b,4/2); constant(a,b)",
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
			expr: "-1+(-a)+(+5)+(+2+3+1); constant(a)",
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
			expr: "pow(9,9)*4*(-3+3)*0+12.3*0-wer*0-0*wed; constant(wer,wed)",
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
		// matrix
		{
			expr: "matrix(2+5,1,1)",
			out:  "matrix(7.000,1.000,1.000)",
		},
		{
			expr: "matrix(2+5,1,1)*matrix(1-2,1,1)",
			out:  "matrix(-7.000,1.000,1.000)",
		},
		{
			expr: "matrix(2+5,9,3, 5-1+0-0,2,2)*matrix(1-2,+5,2,1)",
			out:  "matrix(38.000,17.000,2.000,1.000)",
		},
		{
			expr: "transpose(matrix(2+5,9,3, 5-1+0-0,2,2))*matrix(1-2,+5,2,1)",
			out:  "matrix(8.000,11.000,2.000,1.000)",
		},
		{
			expr: "2*matrix(2+5,1,1)",
			out:  "matrix(14.000,1.000,1.000)",
		},
		{
			expr: "matrix(5+2,1,1)*2",
			out:  "matrix(14.000,1.000,1.000)",
		},
		{
			expr: "a*matrix(2+5,1,1)",
			out:  "matrix(7.000*a,1.000,1.000)",
		},
		{
			expr: "matrix(5+2,1,1)*a",
			out:  "matrix(7.000*a,1.000,1.000)",
		},
		{
			expr: "matrix(5+2*a+a,1,1)*a",
			out:  "matrix(5.000*a+2.000*(a*a)+a*a,1.000,1.000)",
		},
		{
			expr: "matrix(5+a,1,1)*a",
			out:  "matrix(5.000*a+a*a,1.000,1.000)",
		},
		{
			expr: "matrix(5+a,4,0,-2*a,2,2)*a",
			out:  "matrix(5.000*a+a*a,4.000*a,0.000,-2.000*(a*a),2.000,2.000)",
		},
		// integral
		{
			expr: "integral(0,x,0,1);variable(x)",
			out:  "0.000",
		},
		{
			expr: "integral(1,x,0,1);variable(x)",
			out:  "1.000",
		},
		{
			expr: "a*integral(1,x,0,1);variable(x)",
			out:  "a",
		},
		{
			expr: "integral(1,x,0,1)*a;variable(x)",
			out:  "a",
		},
		{
			expr: "integral(a,x,0,1);constant(a);variable(x)",
			out:  "a",
		},
		{
			expr: "integral(a+a,x,0,1);constant(a);variable(x)",
			out:  "2.000 * a",
		},
		{
			expr: "integral(a-b,x,0,1);constant(a);constant(b);variable(x)",
			out:  "a-b",
		},

		{
			expr: "integral(0,x,2,4);variable(x)",
			out:  "0.000",
		},
		{
			expr: "integral(1,x,2,4);variable(x)",
			out:  "2.000",
		},
		{
			expr: "a*integral(1,x,2,4);variable(x)",
			out:  "2.000*a",
		},
		{
			expr: "integral(1,x,2,4)*a;variable(x)",
			out:  "2.000*a",
		},
		{
			expr: "integral(a,x,2,4);constant(a);variable(x)",
			out:  "2.000*a",
		},
		{
			expr: "integral(a+a,x,2,4);constant(a);variable(x)",
			out:  "4.000*a",
		},
		{
			expr: "integral(a-b,x,2,4);constant(a);constant(b);variable(x)",
			out:  "2.000*a-2.000*b",
		},

		{
			expr: "integral(a+x,x,0,1);constant(a);variable(x)",
			out:  "0.500+a",
		},
		{
			expr: "integral(a*x,x,0,1);constant(a);variable(x)",
			out:  "0.500*a",
		},
		{
			expr: "integral(pow(x,2),x,0,1);variable(x)",
			out:  "0.333",
		},
		{
			expr: "integral(a*pow(x,2),x,0,1);variable(x);constant(a)",
			out:  "0.333*a",
		},
		{
			expr: "integral(pow(x,2),x,1,2);variable(x)",
			out:  "2.331",
		},

		{
			expr: "integral(a+x,x,2,3);constant(a);variable(x)",
			out:  "2.500+a",
		},
		{
			expr: "integral(a*x,x,2,3);constant(a);variable(x)",
			out:  "2.500*a",
		},
		{
			expr: "integral(a*x*a,x,2,3);constant(a);variable(x)",
			out:  "2.500*(a*a)",
		},
		{
			expr: "integral(pow(x,2),x,2,3);variable(x)",
			out:  "6.327",
		},
		{
			expr: "integral(pow(x,3),x,2,3);variable(x)",
			out:  "16.250",
		},
		{
			expr: "integral(pow(a*x,3),x,2,3);variable(x);constant(a)",
			out:  "16.250*(a*(a*a))",
		},
		{
			expr: "integral(pow(a*x,2),x,2,3);variable(x);constant(a)",
			out:  "6.327*(a*a)",
		},
		{
			expr: "integral(a*pow(x,2),x,2,3);variable(x);constant(a)",
			out:  "6.327*a",
		},
		{
			expr: "integral(a+a*pow(x,2)+pow(x,3)*a,x,2,3);variable(x);constant(a)",
			out:  "23.577 * a",
		},
		{
			expr: "integral(pow(x,2),x,2,3);variable(x)",
			out:  "6.327",
		},
		{
			expr: "integral(x*a*x*a*x*a,x,2,3);variable(x);constant(a)",
			out:  "16.250*(a*(a*a))",
		},
		{
			expr: "integral(-1.000/L*(1.000/L), s, 0.000, 1.000); constant(L)",
			out:  "-1.000 / L * (1.000 / L)",
		},
		{
			expr: "integral(((sin(q))-(sin(q))*s)/r, s, 0.000, 1.000); constant(q); constant(r); variable(s)",
			out:  "sin(q)/r-0.500/r*sin(q)",
		},
		{
			expr: "integral((v*(-1.000/L)+(1.000-s)*sin(q)/r)*(1.000/L), s, 0.000, 1.000); constant(L); constant(q); constant(r);constant(v);",
			out:  "v*(-1.000/L*(1.000/L))+(1.000/L*(sin(q)/r)-1.000/L*(0.500/r*sin(q)))",
		},
		{
			expr: `integral(transpose(matrix(a*s,1,1))*matrix(b*s,1,1)*matrix(c*s,1,1),s, 1, 2);variable(s);constant(a);constant(b);constant(c)`,
			out:  "matrix(3.750*(a*(b*c)),1.000,1.000)",
		},
		{
			expr: "integral((2.000*(sin(q)*s)-3.000*(sin(q)*(s*s)))/r*(1.000/L), s, 0.000, 1.000);constant(q);constant(r);constant(L);",
			out:  "0.500/L*(2.000/r*sin(q))-0.333/L*(3.000/r*sin(q))",
		},
		{
			expr: " integral(s*(6.000/L*(s*(1.000/L))), s, 0.000, 1.000); constant(L);",
			out:  "1.998/L*(1.000/L)",
		},
		{
			expr: "integral(1.000/L*(-1.000/L)+v*(1.000/L*((sin(q)-sin(q)*s)/r)), s, 0.000, 1.000);constant(L,v,a,q,r); variable(s)",
			out:  "-1.000/L*(1.000/L)+(v*(1.000/L*(sin(q)/r))-v*(1.000/L*(0.500/r*sin(q))))",
		},
		{
			expr: "integral((sin(q)-sin(q)*s)/r*(1.000/L), s, 0.000, 1.000); constant(q,r,L)",
			out:  "1.000/L*(sin(q)/r)-1.000/L*(0.500/r*sin(q))",
		},
		{
			expr: "integral(s*sin(q)/r, s, 0.000, 1.000); constant(q,r)",
			out:  "0.500/r*sin(q)",
		},
		{
			expr: "integral(sin(q)/r, s, 0.000, 1.000); constant(q,r)",
			out:  "sin(q)/r",
		},
		{
			expr: "det(matrix(a,b,c,d,2,2))",
			out:  "a*d-b*c",
		},
		{
			expr: "det(matrix(-2,2,-3,-1,1,3,2,0,1,3,3))",
			out:  "18.000",
		},
		{
			expr: "det(matrix(-1,1.5,1,-1,2,2))",
			out:  "-0.500",
		},
		{
			expr: "det(matrix(a,b,c,d,e,f,g,h,i,3,3))",
			out:  "a*(e*i)-a*(f*h)-(b*(d*i)-b*(f*g))+(c*(d*h)-c*(e*g))",
		},
		{
			expr: "inverse(matrix(1,2,3,0,1,4,5,6,0,3,3))",
			out:  "matrix(-24.000,18.000,5.000,20.000,-15.000,-4.000,-5.000,4.000,1.000,3.000,3.000)",
		},
		{
			expr: "inverse(matrix(a,b,c,d,e,f,g,h,i,3,3)); constant(a,b,c,d,e,f,g,h);",
			out:  "matrix(e*(1.000/(a*(e*i)-a*(f*h)-(b*(d*i)-b*(f*g))+(c*(d*h)-c*(e*g)))*i)-f*(1.000/(a*(e*i)-a*(f*h)-(b*(d*i)-b*(f*g))+(c*(d*h)-c*(e*g)))*h),-1.000*(b*(1.000/(a*(e*i)-a*(f*h)-(b*(d*i)-b*(f*g))+(c*(d*h)-c*(e*g)))*i))+c*(1.000/(a*(e*i)-a*(f*h)-(b*(d*i)-b*(f*g))+(c*(d*h)-c*(e*g)))*h),b*(1.000/(a*(e*i)-a*(f*h)-(b*(d*i)-b*(f*g))+(c*(d*h)-c*(e*g)))*f)-c*(1.000/(a*(e*i)-a*(f*h)-(b*(d*i)-b*(f*g))+(c*(d*h)-c*(e*g)))*e),-1.000*(d*(1.000/(a*(e*i)-a*(f*h)-(b*(d*i)-b*(f*g))+(c*(d*h)-c*(e*g)))*i))+f*(1.000/(a*(e*i)-a*(f*h)-(b*(d*i)-b*(f*g))+(c*(d*h)-c*(e*g)))*g),a*(1.000/(a*(e*i)-a*(f*h)-(b*(d*i)-b*(f*g))+(c*(d*h)-c*(e*g)))*i)-c*(1.000/(a*(e*i)-a*(f*h)-(b*(d*i)-b*(f*g))+(c*(d*h)-c*(e*g)))*g),-1.000*(a*(1.000/(a*(e*i)-a*(f*h)-(b*(d*i)-b*(f*g))+(c*(d*h)-c*(e*g)))*f))+c*(1.000/(a*(e*i)-a*(f*h)-(b*(d*i)-b*(f*g))+(c*(d*h)-c*(e*g)))*d),d*(1.000/(a*(e*i)-a*(f*h)-(b*(d*i)-b*(f*g))+(c*(d*h)-c*(e*g)))*h)-e*(1.000/(a*(e*i)-a*(f*h)-(b*(d*i)-b*(f*g))+(c*(d*h)-c*(e*g)))*g),-1.000*(a*(1.000/(a*(e*i)-a*(f*h)-(b*(d*i)-b*(f*g))+(c*(d*h)-c*(e*g)))*h))+b*(1.000/(a*(e*i)-a*(f*h)-(b*(d*i)-b*(f*g))+(c*(d*h)-c*(e*g)))*g),a*(1.000/(a*(e*i)-a*(f*h)-(b*(d*i)-b*(f*g))+(c*(d*h)-c*(e*g)))*e)-b*(1.000/(a*(e*i)-a*(f*h)-(b*(d*i)-b*(f*g))+(c*(d*h)-c*(e*g)))*d),3.000,3.000)",
		},
		{
			expr: "inverse(matrix(a,b,c,d,2,2))",
			out:  "matrix(d*(1.000/(a*d-b*c)),-1.000*(b*(1.000/(a*d-b*c))),-1.000*(c*(1.000/(a*d-b*c))),a*(1.000/(a*d-b*c)),2.000,2.000)",
		},
		{
			expr: `inverse(matrix( 1,l,0, l,0,1, l,1,0, 3,3)); constant(l);`,
			out:  "matrix(-1.000/(-1.000+l*l),0.000,l*(1.000/(-1.000+l*l)),l*(1.000/(-1.000+l*l)),0.000,-1.000/(-1.000+l*l),l*(1.000/(-1.000+l*l)),-1.000/(-1.000+l*l)+l*(1.000/(-1.000+l*l)*l),0.000-l*(1.000/(-1.000+l*l)*l),3.000,3.000)",
		},
		{
			expr: "l*(l*(1.000/l*(1.000/l*l)))",
			out:  "l",
		},
		// {
		// 	expr: "inverse(matrix( 1,0,0,0,0,0, 0,0,1,0,0,0, 0,0,0,1,0,0, 1,l,0,0,0,0, 0,0,1,l,l*l,l*l*l, 0,0,0,1,2*l,3*l*l, 6,6));",
		// 	out:  "",
		// },
		{
			expr: "6.000 / L * (0.333 / L); constant(L)",
			out: "1.998 / L * (1.000 / L)",
		},
	}

	for i := range tcs {
		t.Run(fmt.Sprintf("%d:%s", i, tcs[i].expr), func(t *testing.T) {
			var (
				act string
				err error
			)
			if testing.Verbose() {
				act, err = sm.Sexpr(os.Stdout, tcs[i].expr)
			} else {
				act, err = sm.Sexpr(nil, tcs[i].expr)
			}
			if err != nil {
				t.Fatal(err)
			}
			act = strings.Replace(act, " ", "", -1)
			ec := strings.Replace(tcs[i].out, " ", "", -1)
			if strings.Contains(act, "integral") {
				t.Errorf("found: %s", "integral")
			}
			if act != ec {
				t.Fatalf("Is not same '%s' != '%s'", act, tcs[i].out)
			}

			// TODO:
			// check by inject numbers
			// create list of constants
			// replace constants into number and get the result
			// simplify expression
			// replace constants into number and get the result
			// comparing diff of results
		})
	}
}

func Example() {
	input := "-1+(-a)+(+5)+(+2+3+1)"
	fmt.Fprintf(os.Stdout, "Input : %s\n\n", input)
	eq, err := sm.Sexpr(os.Stdout, input)
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(os.Stdout, "Output : %s\n", eq)
	// Output:
	// Input : -1+(-a)+(+5)+(+2+3+1)
	//
	// -1.000 - a + 5.000 + (2.000 + 3 + 1)
	// 5.000 + (-1.000 - a) + (2.000 + 3 + 1)
	// 4.000 - a + (2.000 + 3 + 1)
	// 4.000 - a + (2.000 + 3.000 + 1)
	// 4.000 - a + (5.000 + 1)
	// 4.000 - a + (5.000 + 1.000)
	// 4.000 - a + 6.000
	// 6.000 + (4.000 - a)
	// 10.000 - a
	//
	// Output : 10.000 - a
}

// func ExampleSexpr() {
// 	eq, err := sm.Sexpr(os.Stdout, // nil,
// 		`integral(
// 		transpose(matrix(
// 			-1/L,0,0,
// 			(1-s)*sin(q)/r, (1-3*s*s+2*s*s*s)*cos(q)/r, L*(s-2*s*s+s*s*s)*cos(q),
// 			0, (6-12*s)/(L*L), (4-6*s)/L,
// 			0, (6*s-6*s*s)*sin(q)/(r*L), (-1+4*s-3*s*s)*sin(q)/r,
// 			4,3))*
// 			matrix(
// 				1, v, 0, 0,
// 				v, 1, 0, 0,
// 				0, 0, t*t/12, v*t*t/12,
// 				0, 0, v*t*t/12, t*t/12,
// 			4,4)
// 			*
// 			matrix(
// 				1/L,0,0,
// 				s*sin(q)/r, (3*s*s-2*s*s*s)*cos(q)/r, L*(-s*s+s*s*s)*cos(q)/r,
// 				0, (-6+12*s)/L/L, (2-6*s)/L,
// 				0, (-6*s+6*s*s)*sin(q)/r/L, (2*s-3*s*s)*sin(q)/r,
// 			4,3) ,
// 			s, 0, 1);
// 			variable(s);
// 			constant(q);
// 			constant(L);
// 			constant(v);
// 			constant(t);
// 			`,
// 	)
// 	if err != nil {
// 		fmt.Fprintf(os.Stdout, "%v", err)
// 		return
// 	}
// 	fmt.Fprintf(os.Stdout, "%s\n", eq)
// 	fmt.Fprintf(os.Stdout, "integral = %d\n", strings.Count(eq, "integral"))
// 	// Output:
// }
