package auth_test

import (
	"context"
	"testing"

	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/storage"
	"github.com/stretchr/testify/require"
)

func TestRoleTitlePreservesMembership(t *testing.T) {
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		ctx := context.Background()
		repo := auth.NewRepo(db)
		require.NoError(t, repo.EnsureSchema(ctx))
		require.NoError(t, repo.EnsureRolesSchema(ctx))
		require.NoError(t, repo.EnsureRolesSchema(ctx))
		role, err := auth.ParseRole([]byte("name: Cashier\npermissions: {documents: {Report: [read]}}\n"))
		require.NoError(t, err)
		require.NoError(t, repo.SyncRoles(ctx, []*auth.Role{role}))
		initialID := role.ID
		require.NotEmpty(t, initialID)
		user, err := repo.Create(ctx, "cashier-fixture", "test-safe-password", "Fixture", false)
		require.NoError(t, err)
		require.NoError(t, repo.AssignRole(ctx, user.ID, initialID))

		role.Title = "Касса"
		require.NoError(t, repo.SyncRoles(ctx, []*auth.Role{role}))
		require.Equal(t, initialID, role.ID)
		assigned, err := repo.GetRolesForUser(ctx, user.ID)
		require.NoError(t, err)
		require.Len(t, assigned, 1)
		require.Equal(t, "Cashier", assigned[0].Name)
		require.Equal(t, "Касса", assigned[0].DisplayName())
		require.Equal(t, []string{"read"}, assigned[0].Permissions.Documents["Report"])
		roles, err := repo.ListRoles(ctx)
		require.NoError(t, err)
		require.Len(t, roles, 1)
		require.Equal(t, "Касса", roles[0].DisplayName())

		role.Title = ""
		require.NoError(t, repo.SyncRoles(ctx, []*auth.Role{role}))
		assigned, err = repo.GetRolesForUser(ctx, user.ID)
		require.NoError(t, err)
		require.Len(t, assigned, 1)
		require.Equal(t, "Cashier", assigned[0].DisplayName())
	})
}
