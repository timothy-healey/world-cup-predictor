package claudec

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/timhealey/world-cup-predictor/backend/internal/trace"
)

// We test by writing a fake `claude` shell script that echoes the JSON fixture.
func TestDriverParsesStructuredOutput(t *testing.T) {
	tmp := t.TempDir()
	fakeBin := filepath.Join(tmp, "claude")
	body, _ := os.ReadFile("../../testdata/claude-prediction.json")
	script := "#!/bin/sh\ncat <<'EOF'\n" + string(body) + "\nEOF\n"
	require.NoError(t, os.WriteFile(fakeBin, []byte(script), 0o755))

	d := NewDriver(fakeBin, "claude-opus-4-7")
	out, err := d.Predict(context.Background(), "the prompt")
	require.NoError(t, err)
	require.Equal(t, "ARG", out.Winner)
	require.Equal(t, "2-0", out.PredictedScore)
	require.InDelta(t, 0.71, out.WinProbability, 0.001)
	require.Len(t, out.Reasoning, 3)
}

func TestDriverRetriesOnMalformedJSON(t *testing.T) {
	tmp := t.TempDir()
	fakeBin := filepath.Join(tmp, "claude")
	// First invocation returns malformed; second returns valid.
	body, _ := os.ReadFile("../../testdata/claude-prediction.json")
	script := `#!/bin/sh
COUNTER_FILE="` + tmp + `/counter"
if [ ! -f "$COUNTER_FILE" ]; then
  echo 1 > "$COUNTER_FILE"
  echo "not json at all"
  exit 0
fi
cat <<'EOF'
` + string(body) + `
EOF
`
	require.NoError(t, os.WriteFile(fakeBin, []byte(script), 0o755))

	d := NewDriver(fakeBin, "claude-opus-4-7")
	out, err := d.Predict(context.Background(), "prompt")
	require.NoError(t, err)
	require.Equal(t, "ARG", out.Winner)
}

func TestPredictEmitsWirelogLines(t *testing.T) {
	tmp := t.TempDir()
	fake := filepath.Join(tmp, "claude")
	require.NoError(t, os.WriteFile(fake, []byte(`#!/bin/sh
cat <<'EOF'
{"winner":"ARG","predicted_score":"1-0","win_probability":0.5,"reasoning":["x"]}
EOF
`), 0o755))

	var buf bytes.Buffer
	prev := trace.SetWriter(&buf)
	defer trace.SetWriter(prev)

	d := NewDriver(fake, "test-model")
	_, err := d.Predict(t.Context(), "some prompt")
	require.NoError(t, err)

	out := buf.String()
	require.Contains(t, out, "[wcp:predict] → claude -p (prompt: 11 chars)")
	require.Contains(t, out, "[wcp:predict] ✓ ok")
}

func TestPredictEmitsWirelogErrorOnMalformedJSON(t *testing.T) {
	tmp := t.TempDir()
	fake := filepath.Join(tmp, "claude")
	// Always emit garbage so both the initial invoke and the retry fail.
	require.NoError(t, os.WriteFile(fake, []byte(`#!/bin/sh
echo "not json at all"
`), 0o755))

	var buf bytes.Buffer
	prev := trace.SetWriter(&buf)
	defer trace.SetWriter(prev)

	d := NewDriver(fake, "test-model")
	_, err := d.Predict(t.Context(), "p")
	require.Error(t, err)

	out := buf.String()
	// Two invocations happen (initial + retry); we should see at least two
	// start lines and at least one error line for the malformed JSON.
	require.GreaterOrEqual(t, strings.Count(out, "[wcp:predict] → claude -p"), 2)
	require.Contains(t, out, "[wcp:predict] ✗ failed after")
}
