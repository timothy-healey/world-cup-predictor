package odds

import "testing"

func TestOddsAPIName(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"Bosnia-Herzegovina", "Bosnia & Herzegovina"},
		{"Cape Verde Islands", "Cape Verde"},
		{"Congo DR", "DR Congo"},
		{"Czechia", "Czech Republic"},
		{"United States", "USA"},
		{"Mexico", "Mexico"},
		{"", ""},
	}
	for _, c := range cases {
		if got := oddsAPIName(c.in); got != c.want {
			t.Errorf("oddsAPIName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
