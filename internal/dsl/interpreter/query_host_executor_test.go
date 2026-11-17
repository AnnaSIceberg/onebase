package interpreter_test

import (
	"context"
	"testing"

	"github.com/ivantit66/onebase/internal/dsl/interpreter"
	"github.com/ivantit66/onebase/internal/query"
	"github.com/stretchr/testify/require"
)

func TestQueryFactoryHostExecutorKeepsLiveContextAndGuard(t *testing.T) {
	source := &mutableSource{ctx: taggedCtx("before")}
	db := &ctxSpyDB{t: t}
	compiled, executed, guarded := 0, 0, 0
	compiler := func(ctx context.Context, _ string, _ map[string]any) (query.Result, error) {
		require.Equal(t, "after", ctx.Value(ctxTag{}))
		compiled++
		return query.Result{SQL: "SELECT source", Args: []any{"bound"}}, nil
	}
	executor := func(ctx context.Context, res query.Result) ([]map[string]any, error) {
		require.Equal(t, "after", ctx.Value(ctxTag{}))
		require.Equal(t, "SELECT source", res.SQL)
		require.Equal(t, []any{"bound"}, res.Args)
		executed++
		return []map[string]any{{"Код": "raw"}}, nil
	}
	guard := func(ctx context.Context, _ query.Result, rows []map[string]any) (interpreter.GuardedColumns, error) {
		require.Equal(t, "after", ctx.Value(ctxTag{}))
		require.Len(t, rows, 1)
		rows[0]["Код"] = "guarded"
		guarded++
		return nil, nil
	}
	factory := interpreter.NewQueryFactoryGuardedRunnerSource(source, db, &stubReg{}, compiler, guard, executor)
	object := factory(nil).(queryObject)
	source.switchTo(taggedCtx("after"))
	rows := runQuery(t, object).Iterate()
	require.Len(t, rows, 1)
	require.Equal(t, "guarded", rows[0].(*interpreter.Struct).Get("Код"))
	require.Equal(t, 1, compiled)
	require.Equal(t, 1, executed)
	require.Equal(t, 1, guarded)
	require.Empty(t, db.ctxs, "the unbounded default SQL reader must not execute")
}
