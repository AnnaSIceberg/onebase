package storage

import (
	"context"
	"github.com/ivantit66/onebase/internal/metadata"
	"sort"
)

func (db *DB) prepareEntityLegacyIdents(ctx context.Context, entities []*metadata.Entity) error {
	specs := longIdentTables(entities, nil, nil, nil)
	var logicals, physical []string
	for _, spec := range specs {
		logicals = append(logicals, spec.logical)
		physical = append(physical, metadata.SQLIdent(spec.logical))
		physical = append(physical, db.physicalLegacyNames(metadata.LegacySQLIdents(spec.logical))...)
	}
	physical = uniqueStrings(physical)
	sort.Strings(physical)
	return db.WithTxScope(ctx, func(ctx context.Context) error {
		for _, table := range physical {
			exists, err := db.TableExists(ctx, table)
			if err != nil {
				return err
			}
			if exists {
				if err := db.lockEntityTable(ctx, table); err != nil {
					return err
				}
			}
		}
		if err := db.CheckLongIdents(ctx, entities, nil, nil, nil); err != nil {
			return err
		}
		claims := legacyTableClaims(logicals)
		// All tables precede columns: FK targets must already have current names.
		for _, spec := range specs {
			if err := db.renameLegacyTableClaimed(ctx, spec.logical, claims); err != nil {
				return err
			}
		}
		for _, spec := range specs {
			if err := db.renameLegacyColumns(ctx, metadata.SQLIdent(spec.logical), spec.columns); err != nil {
				return err
			}
		}
		return nil
	})
}
