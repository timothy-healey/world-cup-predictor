package predict

type Inputs struct {
	LineupOK        bool
	LineupConfirmed bool // only relevant if LineupOK
	OddsOK          bool
	NewsOK          bool
	ContextOK       bool
}

func Confidence(in Inputs) string {
	levels := []string{"high", "medium", "low"}
	start := 2 // low
	if in.LineupOK {
		if in.LineupConfirmed {
			start = 0 // high
		} else {
			start = 1 // medium
		}
	}
	failures := 0
	if !in.OddsOK {
		failures++
	}
	if !in.NewsOK {
		failures++
	}
	if !in.ContextOK {
		failures++
	}
	idx := start + failures
	if idx > 2 {
		idx = 2
	}
	return levels[idx]
}
