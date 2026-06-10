package trace

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// writer is the destination for all wirelog lines. Defaults to os.Stderr;
// SetWriter replaces it for tests. Guarded by writerMu so SetWriter and the
// logging helpers don't race when tests run in parallel.
var (
	writerMu sync.Mutex
	writer   io.Writer = os.Stderr
)

// SetWriter installs w as the wirelog destination and returns the previous one.
// Pattern matches odds.SetWarnWriter — call SetWriter(prev) in a defer.
func SetWriter(w io.Writer) io.Writer {
	writerMu.Lock()
	defer writerMu.Unlock()
	prev := writer
	writer = w
	return prev
}

func emit(format string, args ...any) {
	writerMu.Lock()
	w := writer
	writerMu.Unlock()
	fmt.Fprintf(w, format+"\n", args...)
}

// HTTPStart logs the outgoing request line.
func HTTPStart(ns, method, url string) {
	emit("[wcp:%s] → %s %s", ns, method, url)
}

// HTTPEnd logs the response status, duration, and body byte count.
// 2xx renders ✓; anything else renders ✗.
func HTTPEnd(ns string, status int, duration time.Duration, bytes int) {
	mark := "✓"
	if status < 200 || status >= 300 {
		mark = "✗"
	}
	emit("[wcp:%s] %s %d (%dms, %s)", ns, mark, status, duration.Milliseconds(), formatBytes(bytes))
}

// HTTPError logs a transport-level failure (no HTTP status was ever received).
func HTTPError(ns string, duration time.Duration, err error) {
	emit("[wcp:%s] ✗ failed after %dms: %s", ns, duration.Milliseconds(), err.Error())
}

// SubprocessStart logs a claude -p invocation about to run.
func SubprocessStart(ns string, promptBytes int) {
	emit("[wcp:%s] → claude -p (prompt: %d chars)", ns, promptBytes)
}

// SubprocessEnd logs a successful exit with duration and stdout size.
func SubprocessEnd(ns string, duration time.Duration, outBytes int) {
	emit("[wcp:%s] ✓ ok (%dms, %s)", ns, duration.Milliseconds(), formatBytes(outBytes))
}

// SubprocessError logs a non-zero exit / context deadline / stderr-bearing failure.
func SubprocessError(ns string, duration time.Duration, err error) {
	emit("[wcp:%s] ✗ failed after %dms: %s", ns, duration.Milliseconds(), err.Error())
}

// formatBytes returns a compact size: B for <1024, KB otherwise (integer KB).
func formatBytes(n int) string {
	if n < 1024 {
		return fmt.Sprintf("%dB", n)
	}
	return fmt.Sprintf("%dKB", n/1024)
}
