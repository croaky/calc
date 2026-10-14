package main

import (
	"go/constant"
	"strings"
	"testing"

	"github.com/croaky/is"
)

func TestEval(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"2 + 2", "4"},
		{"(12 + 4) * 3 / 2.5", "19.2"},
		{"7 / 2", "3.5"},
		{"6 / 2", "3"},
		{"1 / 3", "0.3333333333333333"},
		{"0.1 * 3", "0.3"},
		{"0.1 + 0.2", "0.3"},
		{"-5 + +2", "-3"},
		{"7 % 3", "1"},
		{"1 << 10", "1024"},
		{"1 << 70", "1180591620717411303424"},
		{"0xff & 0x0f", "15"},
		{"6 | 1", "7"},
		{"6 ^ 3", "5"},
		{"6 &^ 2", "4"},
		{"^0", "-1"},
		{"1_000_000 * 3", "3000000"},
		{"2.5 * 4", "10"},
		{"1e3", "1000"},
		{"1e-9", "1e-09"},
		{"1e30 + 0.5", "1e+30"},
	}
	for _, tt := range tests {
		v, err := eval(tt.in, nil)
		if err != nil {
			t.Errorf("eval(%q): %v", tt.in, err)
			continue
		}
		if got := format(v); got != tt.want {
			t.Errorf("eval(%q) = %s, want %s", tt.in, got, tt.want)
		}
	}
}

func TestEvalErrors(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"1 / 0", "division by zero"},
		{"1 % 0", "division by zero"},
		{"1.5 % 1", "% needs integers"},
		{"1.5 & 1", "& needs integers"},
		{"1 << 100000", "shift count must be 0 to 10000"},
		{"1 << -1", "shift count must be 0 to 10000"},
		{"x + 1", "unknown name x"},
		{"_ + 1", "_ has no value yet"},
		{"'a'", "'a' is not a number"},
		{"1 == 1", "unsupported operator =="},
		{"f(1)", "unsupported expression"},
		{"2 +", "syntax error"},
	}
	for _, tt := range tests {
		_, err := eval(tt.in, nil)
		if err == nil || err.Error() != tt.want {
			t.Errorf("eval(%q) error = %v, want %s", tt.in, err, tt.want)
		}
	}
}

func TestEvalPrev(t *testing.T) {
	is := is.New(t)
	v, err := eval("_ * 2", constant.MakeInt64(21))
	is.NoErr(err)
	is.Eq(format(v), "42")
}

func TestRun(t *testing.T) {
	is := is.New(t)
	in := "2 + 2\n\n_ * 10\n1 / 0\n_ + 1\nquit\n3 + 3\n"
	var out strings.Builder
	is.NoErr(run(strings.NewReader(in), &out))
	// An error leaves _ as it was, and nothing after quit runs.
	is.Eq(out.String(), "4\n40\nerror: division by zero\n41\n")
}
