package storage

import (
	"context"
	"sync/atomic"
)

type profileProbeKey struct{}
type ProfileProbeState struct{ exec, query, row atomic.Int64 }

func WithProfileProbe(ctx context.Context, p *ProfileProbeState) context.Context {
	return context.WithValue(ctx, profileProbeKey{}, p)
}
func (p *ProfileProbeState) Snapshot() map[string]int64 {
	return map[string]int64{"exec": p.exec.Load(), "query": p.query.Load(), "query_row": p.row.Load()}
}
func profileProbeCount(ctx context.Context, kind string) {
	p, _ := ctx.Value(profileProbeKey{}).(*ProfileProbeState)
	if p == nil {
		return
	}
	switch kind {
	case "exec":
		p.exec.Add(1)
	case "query":
		p.query.Add(1)
	case "row":
		p.row.Add(1)
	}
}
