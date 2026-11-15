package compose

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/report"
	"strings"
)

// ApplySourceFilters filters masked query rows, retaining the ORIGINAL maps and
// typed values for result_proc. It does not resolve references by reading other
// catalogs or expand permissions. Unsupported display-label reference filters
// fail explicitly, rather than selecting an unfiltered input.
func ApplySourceFilters(rows []Row, filters []report.Filter, columns []string) ([]Row, error) {
	if len(filters) == 0 {
		return rows, nil
	}
	if len(filters) > 128 {
		return nil, fmt.Errorf("too many report source filters (maximum 128)")
	}
	fields := make(map[string]bool, len(columns))
	for _, c := range columns {
		fields[strings.ToLower(c)] = true
	}
	normalized := append([]report.Filter(nil), filters...)
	for i := range normalized {
		f := &normalized[i]
		f.Op = strings.ToLower(strings.TrimSpace(f.Op))
		if f.Field == "" || !fields[strings.ToLower(f.Field)] {
			return nil, fmt.Errorf("unknown report source filter field %q", f.Field)
		}
		switch f.Op {
		case "eq", "ne", "gt", "ge", "lt", "le", "contains":
		default:
			return nil, fmt.Errorf("unsupported report source filter operator %q", f.Op)
		}
		if len(f.Value) > report.MaxSourceSettingsBytes {
			return nil, fmt.Errorf("report source filter value is too large")
		}
		if id, err := uuid.Parse(f.Value); err == nil {
			if id == uuid.Nil {
				f.Value = ""
			} else {
				f.Value = id.String()
			}
		}
	}
	out := make([]Row, 0, len(rows))
	for _, row := range rows {
		view := row
		copied := false
		for _, f := range normalized {
			raw, found := lookupRowField(row, f.Field)
			if !found {
				continue
			}
			id, ok := sourceFilterUUID(raw)
			if !ok {
				continue
			}
			if f.Op != "eq" && f.Op != "ne" {
				return nil, fmt.Errorf("reference source filter %q supports eq/ne only", f.Field)
			}
			if f.Value != "" {
				if _, err := uuid.Parse(f.Value); err != nil {
					return nil, fmt.Errorf("reference source filter %q requires UUID, not a display label", f.Field)
				}
			}
			if !copied {
				view = make(Row, len(row))
				for k, v := range row {
					view[k] = v
				}
				copied = true
			}
			value := ""
			if id != uuid.Nil {
				value = id.String()
			}
			for key := range view {
				if strings.EqualFold(key, f.Field) {
					view[key] = value
					break
				}
			}
		}
		keep := true
		for _, f := range normalized {
			if !matchFilter(view, f) {
				keep = false
				break
			}
		}
		if keep {
			out = append(out, row)
		}
	}
	return out, nil
}

func sourceFilterUUID(v any) (uuid.UUID, bool) {
	switch x := v.(type) {
	case uuid.UUID:
		return x, true
	case string:
		id, e := uuid.Parse(x)
		return id, e == nil
	case []byte:
		id, e := uuid.Parse(string(x))
		return id, e == nil
	default:
		return uuid.Nil, false
	}
}
