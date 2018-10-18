package dual

import "testing"

var correctRounds = []struct{ a, correct int }{
	{0, 0},
	{1, 1},
	{2, 2},
	{3, 4},
	{5, 4},
	{8, 8},
	{10, 8},
	{31, 32},
	{33, 32},
	{80, 64},
	{100, 128},
}

func TestRound(t *testing.T) {
	for _, c := range correctRounds {
		if b := round(c.a); b != c.correct {
			t.Errorf("Expected rounding of %v to be %v. Got %v instead", c.a, c.correct, b)
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	if !DefaultConf(5, 5, 5*5+1).IsValid() {
		t.Errorf("Expected Default Config to be correct")
	}
}
