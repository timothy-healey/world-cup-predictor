package odds

import (
	"fmt"
	"io"
	"os"
	"sync"
)

// warnOut is the destination for warning messages. Tests swap it via
// SetWarnWriter to capture output. Defaults to os.Stderr.
var (
	warnMu  sync.Mutex
	warnOut io.Writer = os.Stderr
)

func warn(format string, args ...any) {
	warnMu.Lock()
	defer warnMu.Unlock()
	fmt.Fprintf(warnOut, "[warn] "+format+"\n", args...)
}

// SetWarnWriter redirects warning output. Returns the previous writer so
// tests can restore it.
func SetWarnWriter(w io.Writer) io.Writer {
	warnMu.Lock()
	defer warnMu.Unlock()
	prev := warnOut
	warnOut = w
	return prev
}
