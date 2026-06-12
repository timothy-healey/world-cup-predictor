package predict

import "time"

// SetNowFn replaces the pipeline's clock for tests. Defined in *_test.go
// so it is compiled only under `go test` — production callers cannot see it.
func (p *Pipeline) SetNowFn(fn func() time.Time) { p.nowFn = fn }
