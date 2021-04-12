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
			out:  "a*c-a*d+(b*c-b*d)",
		},
		{
			expr: "(a + b) * (c - d - s); constant(a,b,c,d,s)",
			out:  "a*c-a*d-a*s+(b*c-b*d-b*s)",
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
			out:  "1.000/(a*(a*a))",
		},
		{
			expr: "pow(a,5-3+1)",
			out:  "a * (a * a)",
		},
		{
			expr: "pow(a+1,2)",
			out:  "1.000+2.000*a+a*a",
		},
		{
			expr: "pow(a+b,5-4)",
			out:  "a + b",
		},
		{
			expr: "pow(a+b,4/2); constant(a,b)",
			out:  "a*a+2.000*(a*b)+b*b",
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
			out:  "a*pow(x,-1.000+a)",
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
			out:  "3.000*x*x",
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
			out:  "2.000*(a*pow(x,-1.000+a))",
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
			out:  "d(u,x)/v - u*d(v,x) / (v * v)",
		},
		{
			expr: "d((2*(3*x-4))/(pow(x,2)+1),x);variable(x);",
			out:  "6.000/(1.000+2.000*(x*x)+x*(x*(x*x)))+16.000*x/(1.000+2.000*(x*x)+x*(x*(x*x)))-6.000*(x*x)/(1.000+2.000*(x*x)+x*(x*(x*x)))",
		},
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
			out:  "matrix(5.000*a+3.000*(a*a),1.000,1.000)",
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
			out:  "2.500*a*a",
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
			out:  "16.250*(a*a)*a",
		},
		{
			expr: "integral(pow(a*x,2),x,2,3);variable(x);constant(a)",
			out:  "6.327*a*a",
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
			out:  "16.250*(a*a)*a",
		},
		{
			expr: "integral(-1.000/L*(1.000/L), s, 0.000, 1.000); constant(L)",
			out:  "-1.000 / (L * L)",
		},
		{
			expr: "integral(((sin(q))-(sin(q))*s)/r, s, 0.000, 1.000); constant(q); constant(r); variable(s)",
			out:  "0.500*sin(q)/r",
		},
		{
			expr: "integral((v*(-1.000/L)+(1.000-s)*sin(q)/r)*(1.000/L), s, 0.000, 1.000); constant(L); constant(q); constant(r);constant(v);",
			out:  "-1.000*v/(L*L)+0.500*sin(q)/(L*r)",
		},
		{
			expr: `integral(transpose(matrix(a*s,1,1))*matrix(b*s,1,1)*matrix(c*s,1,1),s, 1, 2);variable(s);constant(a);constant(b);constant(c)`,
			out:  "matrix(3.750*(a*(b*c)),1.000,1.000)",
		},
		{
			expr: "integral((2.000*(sin(q)*s)-3.000*(sin(q)*(s*s)))/r*(1.000/L), s, 0.000, 1.000);constant(q);constant(r);constant(L);",
			out:  "0.001*sin(q)/(L*r)",
		},
		{
			expr: " integral(s*(6.000/L*(s*(1.000/L))), s, 0.000, 1.000); constant(L);",
			out:  "1.998/(L*L)",
		},
		{
			expr: "integral(1.000/L*(-1.000/L)+v*(1.000/L*((sin(q)-sin(q)*s)/r)), s, 0.000, 1.000);constant(L,v,a,q,r); variable(s)",
			out:  "-1.000/(L*L)+0.500*(v*sin(q))/(L*r)",
		},
		{
			expr: "integral((sin(q)-sin(q)*s)/r*(1.000/L), s, 0.000, 1.000); constant(q,r,L)",
			out:  "0.500*sin(q)/(L*r)",
		},
		{
			expr: "integral(s*sin(q)/r, s, 0.000, 1.000); constant(q,r)",
			out:  "0.500*sin(q)/r",
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
			out:  "a*(e*i)-h*(f*a)-(b*(d*i)-g*(f*b))+(c*(d*h)-g*(e*c))",
		},
		{
			expr: "inverse(matrix(1,2,3,0,1,4,5,6,0,3,3))",
			out:  "matrix(-24.000,18.000,5.000,20.000,-15.000,-4.000,-5.000,4.000,1.000,3.000,3.000)",
		},
		{
			expr: "inverse(matrix(a,b,c,d,e,f,g,h,i,3,3)); constant(a,b,c,d,e,f,g,h);",
			out:  "matrix(e*i/(a*(e*i)-a*(f*h)-(b*(d*i)-b*(f*g))+(c*(d*h)-c*(e*g)))-f*h/(a*(e*i)-a*(f*h)-(b*(d*i)-b*(f*g))+(c*(d*h)-c*(e*g))),-1.000*(b*i)/(a*(e*i)-a*(f*h)-(b*(d*i)-b*(f*g))+(c*(d*h)-c*(e*g)))+c*h/(a*(e*i)-a*(f*h)-(b*(d*i)-b*(f*g))+(c*(d*h)-c*(e*g))),b*f/(a*(e*i)-a*(f*h)-(b*(d*i)-b*(f*g))+(c*(d*h)-c*(e*g)))-c*e/(a*(e*i)-a*(f*h)-(b*(d*i)-b*(f*g))+(c*(d*h)-c*(e*g))),-1.000*(d*i)/(a*(e*i)-a*(f*h)-(b*(d*i)-b*(f*g))+(c*(d*h)-c*(e*g)))+f*g/(a*(e*i)-a*(f*h)-(b*(d*i)-b*(f*g))+(c*(d*h)-c*(e*g))),a*i/(a*(e*i)-a*(f*h)-(b*(d*i)-b*(f*g))+(c*(d*h)-c*(e*g)))-c*g/(a*(e*i)-a*(f*h)-(b*(d*i)-b*(f*g))+(c*(d*h)-c*(e*g))),-1.000*(a*f)/(a*(e*i)-a*(f*h)-(b*(d*i)-b*(f*g))+(c*(d*h)-c*(e*g)))+c*d/(a*(e*i)-a*(f*h)-(b*(d*i)-b*(f*g))+(c*(d*h)-c*(e*g))),d*h/(a*(e*i)-a*(f*h)-(b*(d*i)-b*(f*g))+(c*(d*h)-c*(e*g)))-e*g/(a*(e*i)-a*(f*h)-(b*(d*i)-b*(f*g))+(c*(d*h)-c*(e*g))),-1.000*(a*h)/(a*(e*i)-a*(f*h)-(b*(d*i)-b*(f*g))+(c*(d*h)-c*(e*g)))+b*g/(a*(e*i)-a*(f*h)-(b*(d*i)-b*(f*g))+(c*(d*h)-c*(e*g))),a*e/(a*(e*i)-a*(f*h)-(b*(d*i)-b*(f*g))+(c*(d*h)-c*(e*g)))-b*d/(a*(e*i)-a*(f*h)-(b*(d*i)-b*(f*g))+(c*(d*h)-c*(e*g))),3.000,3.000)",
		},
		{
			expr: "inverse(matrix(a,b,c,d,2,2))",
			out:  "matrix(d/(a*d-b*c),-1.000*b/(a*d-b*c),-1.000*c/(a*d-b*c),a/(a*d-b*c),2.000,2.000)",
		},
		{
			expr: `inverse(matrix( 1,l,0, l,0,1, l,1,0, 3,3)); constant(l);`,
			out:  "matrix(-1.000/(-1.000+l*l),0.000,l/(-1.000+l*l),l/(-1.000+l*l),0.000,-1.000/(-1.000+l*l),l/(-1.000+l*l),1.000,-(l*l/(-1.000+l*l)),3.000,3.000)",
		},
		{
			expr: "l*(l*(1.000/l*(1.000/l*l)))",
			out:  "l",
		},
		{
			expr: "inverse(matrix( 1,0,0,0,0,0, 0,0,1,0,0,0, 0,0,0,1,0,0, 1,l,0,0,0,0, 0,0,1,l,l*l,l*l*l, 0,0,0,1,2*l,3*l*l, 6,6));",
			out:  "matrix(1.000,0.000,0.000,0.000,0.000,0.000,-1.000/l,0.000,0.000,1.000/l,0.000,0.000,0.000,1.000,0.000,0.000,0.000,0.000,0.000,0.000,1.000,0.000,0.000,0.000,0.000,-3.000/(l*l),-2.000/l,0.000,3.000/(l*l),-1.000/l,0.000,2.000/(l*(l*l)),1.000/(l*l),0.000,-2.000/(l*(l*l)),1.000/(l*l),6.000,6.000)",
		},
		{
			expr: "6.000 / L * (0.333 / L); constant(L)",
			out:  "1.998 / (L * L)",
		},
		{
			expr: "(l*(1.000/(l*(l*(l*l)))))",
			out:  "1.000/(l*(l*l))",
		},
		{
			expr: "integral(3.000/(l*(l*(l*l)))*(l*(l*(3.000/(l*(l*(l*l)))*(l*l)))), x, 0.000, l)",
			out:  "9.000/(l*(l*l))",
		},
		{
			expr: "integral(x*(2.000/(l*(l*l))*(3.000/(l*(l*(l*l)))*(l*l))), x, 0.000, l); constant(l);",
			out:  "3.000/(l*(l*l))",
		},
		{
			expr: "36.000/(l*(l*l))*(1.000/(l*l))",
			out:  "36.000/(l*(l*(l*(l*l))))",
		},
		{
			expr: "3.000/(l*(l*(l*l)))*(l*l)",
			out:  "3.000/(l*l)",
		},
		{
			expr: "3.000/(l*(l*(l*l)))*(l*l)+1",
			out:  "1.000 + 3.000/(l*l)",
		},
		{
			expr: "36.000*EJ/(l*(l*l))+(0.000-72.000*(EJ*integral(x/1.000, x, 0.000, l))/(l*(l*(l*(l*l)))))+(0.000-72.000*(EJ*integral(x/1.000, x, 0.000, l))/(l*(l*(l*(l*l))))+47.952*EJ/(l*(l*l))); variable(x); constant(l);",
			out:  "11.952*EJ/(l*(l*l))",
		},
		{
			expr: "0.00+ 1/(36.000/(l*(l*l))*(1.000/(l*l)))-0.00",
			out:  "0.028*(l*(l*(l*(l*l))))",
		},
		{
			expr: "1/(36.000*EJ/(l*(l*l))+(0.000-72.000*(EJ*integral(x/1.000, x, 0.000, l))/(l*(l*(l*(l*l)))))+(0.000-72.000*(EJ*integral(x/1.000, x, 0.000, l))/(l*(l*(l*(l*l))))+47.952*EJ/(l*(l*l)))); variable(x); constant(l);",
			out:  "0.084*(l*(l*l))/EJ",
		},
		{
			expr: "-5*x/y+2*x-0+0+5*y+3*x-1*y+12*x/y+0-0+0-0",
			out:  "5.000*x+4.000*y+7.000*x/y",
		},
		{
			expr: "integral((4.000000*(x*x)/(l*l)+-12.000000*(x*(x*x))/(l*(l*l))+9.000000*(x*(x*(x*x)))/(l*(l*(l*l)))), x, 0.000000, l);constant(l)",
			out:  "0.132*l",
		},
		{
			expr: "x*((0.000000-3.000000*(l*l))/(l*(l*(l*l))))",
			out:  "-3.000*x/(l*l)",
		},
		{
			expr: "-(36.000 * EJ / (l * (l * l))) + 47.952*EJ/(l*(l*l))",
			out:  "11.952*EJ/(l*(l*l))",
		},
		{
			expr: `
			inverse(matrix(
				1,0,0,0,0,0,
				1,1,0,0,0,0,
				1,1,1,0,1,0,
				1,1,1,1,0,0,
				1,1,1,1,1,0,
				1,1,1,1,1,1,
			6,6))`,
			out: "matrix(1.000,-0.000,0.000,-0.000,0.000,-0.000,-1.000,1.000,-0.000,0.000,-0.000,0.000,0.000,-1.000,1.000,1.000,-1.000,-0.000,-0.000,0.000,-1.000,0.000,1.000,0.000,0.000,-0.000,0.000,-1.000,1.000,-0.000,-0.000,0.000,-0.000,0.000,-1.000,1.000,6.000,6.000)",
		},
		{
			expr: "-2.99997*(EJ/l)/(-2.99997*EJ)",
			out:  "1.000/l",
		},
		{
			expr: "matrix(-2.99997*(EJ/(l*l)), 0.00000, 2.99997*(EJ/(l*l)), 2.99997*(EJ/l), 4.00000, 1.00000)/(-1*(2.99997 * (EJ / l)))",
			out:  "matrix(0.999/l,0.000,-0.999/l,-0.999,4.000,1.000)",
		},
		{
			expr: "2.00000*(l*l)-l*l",
			out:  "l*l",
		},
		{
			expr: "L*L*det(matrix(A*E/L,0,0,0,4*E*J/L+2*P*L/15,-(6*E*J/(L*L)+P/10),0,-2*(6*E*J/(L*L)+P/10),2*(12*E*J/(L*L*L)+6*P/2/L),3,3))",
			out:  "96.000*(E*(J*(J*(E*(E*A)))))/(L*(L*L))+3.192*(E*(P*(J*(E*A))))/L-72.000*(E*(J*(E*(J*(A*E)))))/(L*(L*L))+21.600*(E*(J*(P*(E*A))))/L+0.778*(E*(P*(P*(A*L))))",
		},
		{
			expr: "(72.000*(A*(E*(E*(J*(E*J)))))+1.200*(L*(L*(A*(E*(E*(J*P)))))))/(A*(E*(E*(J*(L*(L*L))))))",
			out:  "72.000*(E*J)/(L*(L*L))+1.200*P/L",
		},
		{
			expr: "-11.99952*EJ/(-11.99952*EJ+-144.00000*EJ/(l*l)) + -144.00000*EJ/(-11.99952*(l*(l*EJ))+-144.00000*EJ)",
			out:  "-12.000*EJ/(-12.000*EJ-144.000*EJ/(l*l))-144.000*EJ/(-12.000*(l*(l*EJ))-144.000*EJ)",
		},
		{
			expr: "matrix(1,2,3,4,2,2)+matrix(3,4,5,6,2,2)",
			out:  "matrix(4.000,6.000,8.000,10.000,2.000,2.000)",
		},
		{
			expr: "matrix(1,2,1,4,2,2)-matrix(3,4,5,6,2,2)",
			out:  "matrix(-2.000,-2.000,-4.000,-2.000,2.000,2.000)",
		},
		{
			expr: "d(-(6.00000*x/(l*l)), x); variable(x); constant(l);",
			out:  "-6.000 / (l * l)",
		},
		{
			expr: "integral(pow(0.5*pow(q1*x-q2*x/L+q3*x*x/L/L,2),2),x,0,L);constant(q1,q2,q3,L);variable(x);",
			out:  "0.050*(L*(L*(L*(L*(L*(q1*(q1*(q1*q1))))))))-0.200*(L*(L*(L*(L*(q1*(q1*(q1*q2)))))))+0.300*(L*(L*(L*(q1*(q1*(q2*q2))))))-0.200*(L*(L*(q1*(q2*(q2*q2)))))+0.050*(L*(q2*(q2*(q2*q2))))+0.167*(L*(L*(L*(L*(q1*(q1*(q1*q3)))))))-0.501*(L*(L*(L*(q1*(q1*(q2*q3))))))+(0.501*(L*(L*(q1*(q2*(q2*q3)))))-0.167*(L*(q2*(q2*(q2*q3))))+(0.214*(L*(L*(L*(q1*(q1*(q3*q3))))))-0.429*(L*(L*(q1*(q2*(q3*q3))))))+(0.214*(L*(q2*(q2*(q3*q3))))+0.125*(L*(L*(q1*(q3*(q3*q3)))))-0.125*(L*(q2*(q3*(q3*q3))))+0.028*(L*(q3*(q3*(q3*q3))))))",
		},
		{
			expr: "a*(b+c)",
			out:  "a*b+a*c",
		},
		{
			expr: "(b+c)*a",
			out:  "b*a+c*a",
		},
		{
			expr: "d(matrix(EA*(q1*q4)/(2.00000*L)-EA*(q1*q1/L)/2.00000, 1.00000, 1.00000),q1)/q1;constant(L,x,EA); variable(q1);",
			out:  "matrix(0.500*(EA*q4)/(L*q1)-EA/L,1.000,1.000)",
		},
		{
			expr: "(-1.000*(L*EA) - -0.000)",
			out:  "-1.000 * (L * EA)",
		},
		{
			expr: "integral(d(matrix(-1*(EA*q4)/L+EA*q1/L, 1, 1),q4),q4,0,q4);constant(L,x); variable(q4)",
			out:  "matrix(-1.000*(EA*q4)/L,1.000,1.000)",
		},
		{
			expr: "integral(d(EA*q1/L-EA*q4/L,q1),q1,0,q1);constant(EA,L,x); variable(q1)",
			out:  "EA*q1/L",
		},
		{
			expr: "-18.00000*(EA*(q5*(q5*(q6*q2)))/(L*(L*(L*(L*(L*L))))));constant(q2,q5,q6,L)",
			out:  "-18.000*(q2*(q5*(q5*(q6*EA)))/(L*(L*(L*(L*(L*L))))))",
		},
		// 		{
		// 			expr: `
		// d(0.50000*(q1*(q1*EA))/L - 0.50000*(q1*(q4*EA))/L - 0.50000*(q1*(EA*q4))/L + 0.50000*(q4*(q4*EA))/L - 12.00000*(q1*(EA*(q3*q2)))/(L*(L*L)) + 18.00000*(q2*(q5*(q1*EA)))/(L*(L*(L*L))) - 6.00000*(q1*(q6*(EA*q2)))/(L*(L*L)) - 12.00000*(q1*(q2*(EA*q3)))/(L*(L*L)) - 8.00000*(q1*(EA*(q3*q3)))/(L*L) + 12.00000*(q3*(q5*(q1*EA)))/(L*(L*L)) - 4.00000*(q1*(q6*(EA*q3)))/(L*L) + 12.00000*(q5*(EA*(q1*q3)))/(L*(L*L)) - 18.00000*(q1*(q5*(EA*q5)))/(L*(L*(L*L))) + 6.00000*(q5*(q6*(q1*EA)))/(L*(L*L)) - 6.00000*(q1*(q2*(EA*q6)))/(L*(L*L)) - 4.00000*(q1*(EA*(q3*q6)))/(L*L) + 6.00000*(q6*(q5*(q1*EA)))/(L*(L*L)) - 2.00000*(EA*(q1*(q6*q6)))/(L*L) + 12.00000*(q3*(q2*(q4*EA)))/(L*(L*L)) - 18.00000*(q4*(EA*(q2*q5)))/(L*(L*(L*L))) + 6.00000*(EA*(q2*(q4*q6)))/(L*(L*L)) + 8.00000*(q3*(q3*(q4*EA)))/(L*L) - 12.00000*(q4*(EA*(q3*q5)))/(L*(L*L)) + 4.00000*(EA*(q3*(q4*q6)))/(L*L) - 18.00000*(q4*(EA*(q5*q2)))/(L*(L*(L*L))) - 12.00000*(q4*(q3*(q5*EA)))/(L*(L*L)) + 18.00000*(EA*(q5*(q4*q5)))/(L*(L*(L*L))) - 6.00000*(q4*(EA*(q5*q6)))/(L*(L*L)) + 4.00000*(q3*(q6*(q4*EA)))/(L*L) - 6.00000*(q4*(EA*(q6*q5)))/(L*(L*L)) + 18.00000*(EA*(q2*(q1*q3)))/(L*(L*L)) - 36.00000*(EA*(q5*(q2*q1)))/(L*(L*(L*L))) + 18.00000*(EA*(q2*(q1*q6)))/(L*(L*L)) + 24.00000*(q3*(q2*(q1*EA)))/(L*(L*L)) + 12.00000*(EA*(q3*(q1*q3)))/(L*L) - 24.00000*(EA*(q5*(q3*q1)))/(L*(L*L)) + 12.00000*(EA*(q3*(q1*q6)))/(L*L) - 36.00000*(q2*(EA*(q5*q1)))/(L*(L*(L*L))) - 18.00000*(q5*(q3*(EA*q1)))/(L*(L*L)) + 36.00000*(q5*(EA*(q1*q5)))/(L*(L*(L*L))) - 18.00000*(q5*(q6*(EA*q1)))/(L*(L*L)) + 12.00000*(q6*(q2*(q1*EA)))/(L*(L*L)) + 6.00000*(q1*(q6*(q3*EA)))/(L*L) - 12.00000*(EA*(q5*(q6*q1)))/(L*(L*L)) + 6.00000*(q1*(q6*(q6*EA)))/(L*L) - 36.00000*(q2*(EA*(q2*q4)))/(L*(L*(L*L))) - 18.00000*(q3*(EA*(q2*q4)))/(L*(L*L)) - 18.00000*(q6*(EA*(q2*q4)))/(L*(L*L)) - 24.00000*(EA*(q3*(q2*q4)))/(L*(L*L)) - 12.00000*(q3*(EA*(q3*q4)))/(L*L) + 24.00000*(q5*(q3*(q4*EA)))/(L*(L*L)) - 12.00000*(q6*(EA*(q3*q4)))/(L*L) + 36.00000*(EA*(q5*(q4*q2)))/(L*(L*(L*L))) - 36.00000*(q5*(q5*(EA*q4)))/(L*(L*(L*L))) - 12.00000*(EA*(q6*(q2*q4)))/(L*(L*L)) - 6.00000*(q6*(q3*(EA*q4)))/(L*L) + 12.00000*(q5*(q6*(q4*EA)))/(L*(L*L)) - 6.00000*(q6*(q6*(EA*q4)))/(L*L) - 5.99976*(EA*(q2*(q1*q2)))/(L*(L*(L*L))) - 11.99988*(q3*(EA*(q1*q2)))/(L*(L*L)) + 23.99976*(q2*(EA*(q1*q5)))/(L*(L*(L*L))) - 11.99988*(q6*(EA*(q1*q2)))/(L*(L*L)) - 11.99988*(EA*(q3*(q1*q2)))/(L*(L*L)) - 5.99994*(q3*(q3*(EA*q1)))/(L*L) + 11.99988*(q3*(EA*(q1*q5)))/(L*(L*L)) - 5.99994*(q3*(q6*(EA*q1)))/(L*L) + 41.99976*(q5*(q2*(q1*EA)))/(L*(L*(L*L))) + 11.99988*(EA*(q5*(q1*q3)))/(L*(L*L)) - 23.99976*(q5*(q5*(q1*EA)))/(L*(L*(L*L))) + 11.99988*(EA*(q5*(q1*q6)))/(L*(L*L)) - 11.99988*(EA*(q6*(q1*q2)))/(L*(L*L)) - 5.99994*(q6*(q3*(EA*q1)))/(L*L) + 11.99988*(q6*(EA*(q1*q5)))/(L*(L*L)) - 5.99994*(q6*(q6*(EA*q1)))/(L*L) + 41.99976*(EA*(q2*(q4*q2)))/(L*(L*(L*L))) + 11.99988*(q3*(EA*(q4*q2)))/(L*(L*L)) - 23.99976*(q2*(EA*(q4*q5)))/(L*(L*(L*L))) + 11.99988*(q6*(EA*(q4*q2)))/(L*(L*L)) + 23.99988*(EA*(q3*(q4*q2)))/(L*(L*L)) + 5.99994*(q4*(q3*(q3EA*q1/L-EA*q4/L+24.000*(EA*(q3*q5))/(L*(L*L))+6.000*(EA*(q2*q3))/(L*(L*L))+12.000*(EA*(q2*q6))/(L*(L*L))-6.000*(EA*(q2*q2))/(L*(L*(L*L)))-12.000*(EA*(q3*q2))/(L*(L*L))+6.000*(EA*(q2*q5))/(L*(L*(L*L)))-2.000*(EA*(q3*q3))/(L*L)+2.000*(EA*(q3*q6))/(L*L)+6.000*(EA*(q5*q2))/(L*(L*(L*L)))-18.000*(EA*(q5*q3))/(L*(L*L))-6.000*(EA*(q5*q5))/(L*(L*(L*L)))-12.000*(EA*(q5*q6))/(L*(L*L))-18.000*(EA*(q6*q2))/(L*(L*L))-4.000*(EA*(q6*q3))/(L*L)+18.000*(EA*(q6*q5))/(L*(L*L))-2.000*(EA*(q6*q6))/(L*L)+EA*q1/L-EA*q4/L+24.000*(EA*(q3*q5))/(L*(L*L))+6.000*(EA*(q2*q3))/(L*(L*L))+12.000*(EA*(q2*q6))/(L*(L*L))-6.000*(EA*(q2*q2))/(L*(L*(L*L)))-12.000*(EA*(q3*q2))/(L*(L*L))+6.000*(EA*(q2*q5))/(L*(L*(L*L)))-2.000*(EA*(q3*q3))/(L*L)+2.000*(EA*(q3*q6))/(L*L)+6.000*(EA*(q5*q2))/(L*(L*(L*L)))-18.000*(EA*(q5*q3))/(L*(L*L))-6.000*(EA*(q5*q5))/(L*(L*(L*L)))-12.000*(EA*(q5*q6))/(L*(L*L))-18.000*(EA*(q6*q2))/(L*(L*L))-4.000*(EA*(q6*q3))/(L*L)+18.000*(EA*(q6*q5))/(L*(L*L))-2.000*(EA*(q6*q6))/(L*L)*EA)))/(L*L) + 6.00012*(q3*(EA*(q4*q5)))/(L*(L*L)) + 5.99994*(q4*(q3*(q6*EA)))/(L*L) + 12.00024*(q5*(q2*(q4*EA)))/(L*(L*(L*L))) - 11.99988*(EA*(q5*(q4*q3)))/(L*(L*L)) + 23.99976*(q5*(q5*(q4*EA)))/(L*(L*(L*L))) - 11.99988*(EA*(q5*(q4*q6)))/(L*(L*L)) + 17.99988*(EA*(q6*(q4*q2)))/(L*(L*L)) + 5.99994*(q4*(q6*(q3*EA)))/(L*L) + 6.00012*(q6*(EA*(q4*q5)))/(L*(L*L)) + 7.99994*(q4*(q6*(q6*EA)))/(L*L) + (54.00000*(q2*(q2*(q6*(q2*EA))))/(L*(L*(L*(L*(L*L))))) + 36.00000*(q3*(q2*(q6*(q2*EA))))/(L*(L*(L*(L*L)))) - 162.00000*(q2*(q2*(EA*(q5*q2))))/(L*(L*(L*(L*(L*(L*L)))))) - 108.00000*(q2*(q2*(q3*(q5*EA))))/(L*(L*(L*(L*(L*L))))) - 54.00000*(q2*(q2*(EA*(q5*q6))))/(L*(L*(L*(L*(L*L))))) + 36.00000*(q2*(q3*(q6*(q2*EA))))/(L*(L*(L*(L*L)))) + 24.00000*(q3*(q3*(q6*(q2*EA))))/(L*(L*(L*L))) - 108.00000*(q3*(q2*(EA*(q5*q2))))/(L*(L*(L*(L*(L*L))))) + 108.00000*(q5*(q3*(q5*(q2*EA))))/(L*(L*(L*(L*(L*L))))) - 36.00000*(q3*(q2*(EA*(q5*q6))))/(L*(L*(L*(L*L)))) + 162.00000*(q5*(q5*(EA*(q2*q2))))/(L*(L*(L*(L*(L*(L*L)))))) + 108.00000*(q5*(q5*(EA*(q2*q3))))/(L*(L*(L*(L*(L*L))))) + 108.00000*(EA*(q5*(q3*(q2*q5))))/(L*(L*(L*(L*(L*L))))) + 54.00000*(q6*(q5*(EA*(q2*q5))))/(L*(L*(L*(L*(L*L))))) + 54.00000*(q5*(q5*(EA*(q2*q6))))/(L*(L*(L*(L*(L*L))))) + 18.00000*(q2*(q6*(q6*(q2*EA))))/(L*(L*(L*(L*L))))),q1);constant(EA,EJ,L,x); variable(q1)`,
		// 			out: "EA*q1/L-EA*q4/L+6.000*(EA*(q2*q3))/(L*(L*L))+12.000*(EA*(q2*q6))/(L*(L*L))-6.000*(EA*(q2*q2))/(L*(L*(L*L)))+6.000*(EA*(q2*q5))/(L*(L*(L*L)))-12.000*(EA*(q3*q2))/(L*(L*L))-2.000*(EA*(q3*q3))/(L*L)+24.000*(EA*(q3*q5))/(L*(L*L))+2.000*(EA*(q3*q6))/(L*L)+6.000*(EA*(q5*q2))/(L*(L*(L*L)))-18.000*(EA*(q5*q3))/(L*(L*L))-6.000*(EA*(q5*q5))/(L*(L*(L*L)))-12.000*(EA*(q5*q6))/(L*(L*L))-18.000*(EA*(q6*q2))/(L*(L*L))-4.000*(EA*(q6*q3))/(L*L)+18.000*(EA*(q6*q5))/(L*(L*L))-2.000*(EA*(q6*q6))/(L*L)+6.000*(q4*(q3*q3EA))/(L*(L*L))+6.000*(EA*(q4*q3))/(L*(L*L))",
		// 		},
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
			if strings.Contains(act, "+-") {
				t.Errorf("Not valid +- sign confuse")
			}
			if act != ec {
				t.Fatalf("Is not same \nActual : '%s'\nExpect : '%s'", act, tcs[i].out)
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
	// 4.000 + -a + (2.000 + 3 + 1)
	// 4.000 - a + (2.000 + 3 + 1)
	// 4.000 - a + (2.000 + 3.000 + 1)
	// 4.000 - a + (5.000 + 1)
	// 4.000 - a + (5.000 + 1.000)
	// 4.000 - a + 6.000
	// 10.000 + -a
	// 10.000 - a
	//
	// Output : 10.000 - a
}
