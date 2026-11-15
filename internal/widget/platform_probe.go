package widget

// Temporary private-copy diagnostic. Never copy into a shipping branch.
import (
	"context"
	"encoding/json"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/query"
	"os"
	"sync"
	"time"
)

var platformProbeOnce sync.Once

func platformQueryProbe(ctx context.Context, r *Runner, w *metadata.Widget, q *query.Result) func(int) {
	if os.Getenv("PLATFORM_REAL_READY") != "1" || w.Name != "СписокЗадач" {
		return func(int) {}
	}
	started := time.Now()
	var plan []map[string]any
	var sql string
	platformProbeOnce.Do(func() {
		plan, _, _ = r.Store.RunQuery(ctx, "EXPLAIN QUERY PLAN "+q.SQL, q.Args)
		sql = q.SQL
	})
	return func(rows int) {
		raw, _ := json.Marshal(map[string]any{"query_ms": float64(time.Since(started).Microseconds()) / 1000, "rows": rows, "plan": plan, "sql": sql})
		f, err := os.OpenFile("/home/onebase/cc2-release36-dev-profile-snapshot-20261113/platform-private-query-probe.jsonl", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err == nil {
			defer f.Close()
			f.Write(append(raw, '\n'))
		}
	}
}
