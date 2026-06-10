package claudec

import (
	"fmt"
	"strings"
)

type PromptInputs struct {
	SystemPrompt string
	HomeName     string
	AwayName     string
	KickoffUTC   string
	Stage        string

	OddsBlock    string
	NewsBlock    string
	LineupBlock  string
	ContextBlock string
}

func BuildPrompt(in PromptInputs) string {
	var b strings.Builder
	b.WriteString(in.SystemPrompt)
	b.WriteString("\n\n---\n\n")
	fmt.Fprintf(&b, "MATCH: %s vs %s\nKICKOFF (UTC): %s\nSTAGE: %s\n\n",
		in.HomeName, in.AwayName, in.KickoffUTC, in.Stage)

	addBlock(&b, "ODDS", in.OddsBlock)
	addBlock(&b, "NEWS", in.NewsBlock)
	addBlock(&b, "LINEUP", in.LineupBlock)
	addBlock(&b, "TOURNAMENT CONTEXT", in.ContextBlock)
	return b.String()
}

func addBlock(b *strings.Builder, label, body string) {
	if body == "" {
		fmt.Fprintf(b, "%s: (not available)\n\n", label)
		return
	}
	fmt.Fprintf(b, "%s:\n%s\n\n", label, body)
}
