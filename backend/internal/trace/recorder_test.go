package trace

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// fixedNow lets us control timestamps in tests.
func fixedNow(start time.Time, advance time.Duration) func() time.Time {
	calls := 0
	return func() time.Time {
		t := start.Add(time.Duration(calls) * advance)
		calls++
		return t
	}
}

func TestRecorderHappyPath(t *testing.T) {
	r := New()
	r.now = fixedNow(time.Date(2026, 6, 25, 17, 30, 0, 0, time.UTC), 100*time.Millisecond)

	r.Start("odds")
	r.Finish("odds", nil, "bookmaker=william home=1.40")
	r.Start("news")
	r.Finish("news", nil, "h news / a news")
	r.Start("lineup")
	r.Finish("lineup", nil, "confirmed=true notes=")
	r.Start("context")
	r.Finish("context", nil, "Tournament... / Track...")
	r.Start("predict")
	r.Finish("predict", nil, `{"winner":"ARG"}`)

	raw, err := r.JSON()
	require.NoError(t, err)
	var entries []map[string]any
	require.NoError(t, json.Unmarshal(raw, &entries))
	require.Len(t, entries, 5)
	require.Equal(t, "odds", entries[0]["kind"])
	require.Equal(t, "news", entries[1]["kind"])
	require.Equal(t, "lineup", entries[2]["kind"])
	require.Equal(t, "context", entries[3]["kind"])
	require.Equal(t, "predict", entries[4]["kind"])
	require.Equal(t, true, entries[0]["ok"])
	require.Equal(t, "", entries[0]["error"])
	require.Equal(t, float64(100), entries[0]["duration_ms"])
	require.Equal(t, "bookmaker=william home=1.40", entries[0]["snippet"])
	require.Equal(t, "2026-06-25T17:30:00.000Z", entries[0]["started_at"])
}

func TestRecorderErrorIsRecorded(t *testing.T) {
	r := New()
	r.now = fixedNow(time.Date(2026, 6, 25, 17, 30, 0, 0, time.UTC), 380*time.Millisecond)
	r.Start("odds")
	r.Finish("odds", errors.New("no odds found for ARG vs SAU"), "")

	raw, _ := r.JSON()
	var entries []map[string]any
	require.NoError(t, json.Unmarshal(raw, &entries))
	require.Equal(t, false, entries[0]["ok"])
	require.Equal(t, "no odds found for ARG vs SAU", entries[0]["error"])
	require.Equal(t, "", entries[0]["snippet"])
}

func TestRecorderSnippetTruncatedTo400Chars(t *testing.T) {
	r := New()
	r.Start("news")
	long := strings.Repeat("a", 500)
	r.Finish("news", nil, long)
	raw, _ := r.JSON()
	var entries []map[string]any
	require.NoError(t, json.Unmarshal(raw, &entries))
	got := entries[1]["snippet"].(string)
	require.Equal(t, 400, len(got))
}

func TestRecorderSnippetTruncationIsUTF8Safe(t *testing.T) {
	r := New()
	r.Start("news")
	// 200× "ñ" (2 bytes) = 400 bytes, but only 200 runes.
	// 201× "ñ" = 402 bytes, must trim back to 400 bytes WITHOUT splitting a rune.
	s := strings.Repeat("ñ", 201)
	r.Finish("news", nil, s)
	raw, _ := r.JSON()
	var entries []map[string]any
	require.NoError(t, json.Unmarshal(raw, &entries))
	got := entries[1]["snippet"].(string)
	require.LessOrEqual(t, len(got), 400)
	// Truncated string must still be valid UTF-8 (no replacement char appears).
	require.NotContains(t, got, "�")
}

func TestRecorderUnstartedKindMarkedNotRun(t *testing.T) {
	r := New()
	// Start nothing — JSON should still emit five entries with "not run".
	raw, _ := r.JSON()
	var entries []map[string]any
	require.NoError(t, json.Unmarshal(raw, &entries))
	require.Len(t, entries, 5)
	for _, e := range entries {
		require.Equal(t, false, e["ok"])
		require.Equal(t, "not run", e["error"])
		require.Equal(t, float64(0), e["duration_ms"])
	}
}

func TestRecorderFinishWithoutStartPanics(t *testing.T) {
	r := New()
	require.Panics(t, func() { r.Finish("odds", nil, "") })
}

func TestRecorderDoubleFinishPanics(t *testing.T) {
	r := New()
	r.Start("odds")
	r.Finish("odds", nil, "")
	require.Panics(t, func() { r.Finish("odds", nil, "") })
}

func TestRecorderUnknownKindPanics(t *testing.T) {
	r := New()
	require.Panics(t, func() { r.Start("bogus") })
}
