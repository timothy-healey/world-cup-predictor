package claudec

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
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
