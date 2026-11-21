package query

import (
	"fmt"

	"github.com/ivantit66/onebase/internal/storage"
)

func isISODateTimeCall(tokens []tok, at int) bool {
	if at+1 >= len(tokens) || tokens[at].kind != tIdent || tokens[at+1].kind != tLParen {
		return false
	}
	name := lowerFast(tokens[at].val)
	return name == "датаизiso" || name == "isodatetime"
}

func validateISODateTimeCalls(tokens []tok) error {
	for i := range tokens {
		if !isISODateTimeCall(tokens, i) {
			continue
		}
		end := scalarCallEnd(tokens, i)
		if end < i+3 || len(splitTopLevelArgs(tokens[i+2:end])) != 1 {
			return fmt.Errorf("ДатаИзISO: ожидается один аргумент")
		}
	}
	return nil
}

// PostgreSQL 14+ JSON path can test and parse a row value using silent mode.
// Convert the parsed JSON string back to a timestamp; a failed parse stays NULL.
func postgresISODateTimeRewrite() funcRewrite {
	prefix := "CAST(btrim(CAST(jsonb_path_query_first(to_jsonb(replace(CAST("
	path := `$ ? (@ like_regex "` + storage.ISODateTimePattern + `" && !(@ like_regex "^0000-")).datetime()`
	suffix := ` AS TEXT), 'Z', '+00:00')), '` + path + `', '{}', true) AS TEXT), '"') AS TIMESTAMPTZ)`
	return funcRewrite{prefix: tokenizeFragment(prefix), suffix: tokenizeFragment(suffix)}
}
