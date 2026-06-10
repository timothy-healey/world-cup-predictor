package trace

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHTTPStartEmitsArrowLine(t *testing.T) {
	var buf bytes.Buffer
	prev := SetWriter(&buf)
	defer SetWriter(prev)

	HTTPStart("odds", "GET", "https://api.the-odds-api.com/v4/sports/soccer_fifa_world_cup/odds/?regions=uk")
	require.Equal(t,
		"[wcp:odds] → GET https://api.the-odds-api.com/v4/sports/soccer_fifa_world_cup/odds/?regions=uk\n",
		buf.String(),
	)
}

func TestHTTPEndEmitsCheckLineWithStatusDurationBytes(t *testing.T) {
	var buf bytes.Buffer
	prev := SetWriter(&buf)
	defer SetWriter(prev)

	HTTPEnd("odds", 200, 412*time.Millisecond, 14336)
	require.Equal(t, "[wcp:odds] ✓ 200 (412ms, 14KB)\n", buf.String())
}

func TestHTTPEndNon2xxEmitsCross(t *testing.T) {
	var buf bytes.Buffer
	prev := SetWriter(&buf)
	defer SetWriter(prev)

	HTTPEnd("odds", 429, 50*time.Millisecond, 128)
	require.Equal(t, "[wcp:odds] ✗ 429 (50ms, 128B)\n", buf.String())
}

func TestHTTPErrorEmitsCrossLine(t *testing.T) {
	var buf bytes.Buffer
	prev := SetWriter(&buf)
	defer SetWriter(prev)

	HTTPError("odds", 15*time.Second, errors.New("dial tcp: i/o timeout"))
	require.Equal(t, "[wcp:odds] ✗ failed after 15000ms: dial tcp: i/o timeout\n", buf.String())
}

func TestSubprocessStartEmitsPromptSize(t *testing.T) {
	var buf bytes.Buffer
	prev := SetWriter(&buf)
	defer SetWriter(prev)

	SubprocessStart("news", 287)
	require.Equal(t, "[wcp:news] → claude -p (prompt: 287 chars)\n", buf.String())
}

func TestSubprocessEndEmitsDurationAndOutputSize(t *testing.T) {
	var buf bytes.Buffer
	prev := SetWriter(&buf)
	defer SetWriter(prev)

	SubprocessEnd("news", 3614*time.Millisecond, 1024)
	require.Equal(t, "[wcp:news] ✓ ok (3614ms, 1KB)\n", buf.String())
}

func TestSubprocessErrorEmitsDurationAndError(t *testing.T) {
	var buf bytes.Buffer
	prev := SetWriter(&buf)
	defer SetWriter(prev)

	SubprocessError("news", 91240*time.Millisecond, errors.New("context deadline exceeded"))
	require.Equal(t, "[wcp:news] ✗ failed after 91240ms: context deadline exceeded\n", buf.String())
}

func TestFormatBytesHumanReadable(t *testing.T) {
	cases := []struct {
		in   int
		want string
	}{
		{0, "0B"},
		{1, "1B"},
		{999, "999B"},
		{1000, "1000B"},
		{1024, "1KB"},
		{14336, "14KB"},
		{1048576, "1024KB"}, // 1MB but we only go to KB
	}
	for _, c := range cases {
		require.Equal(t, c.want, formatBytes(c.in), "in=%d", c.in)
	}
}
