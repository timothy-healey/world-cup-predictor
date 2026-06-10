package predict

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfidence(t *testing.T) {
	cases := []struct {
		name string
		in   Inputs
		want string
	}{
		{"all ok + confirmed xi", Inputs{LineupOK: true, LineupConfirmed: true, OddsOK: true, NewsOK: true, ContextOK: true}, "high"},
		{"all ok + squad fallback", Inputs{LineupOK: true, LineupConfirmed: false, OddsOK: true, NewsOK: true, ContextOK: true}, "medium"},
		{"lineup crashed", Inputs{LineupOK: false, OddsOK: true, NewsOK: true, ContextOK: true}, "low"},
		{"start medium, lose odds", Inputs{LineupOK: true, LineupConfirmed: false, OddsOK: false, NewsOK: true, ContextOK: true}, "low"},
		{"start high, lose odds", Inputs{LineupOK: true, LineupConfirmed: true, OddsOK: false, NewsOK: true, ContextOK: true}, "medium"},
		{"start high, lose two", Inputs{LineupOK: true, LineupConfirmed: true, OddsOK: false, NewsOK: false, ContextOK: true}, "low"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.want, Confidence(c.in))
		})
	}
}
