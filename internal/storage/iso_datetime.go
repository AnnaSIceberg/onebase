package storage

import (
	"regexp"
	"time"
)

// ISODateTimePattern is the explicit-zone form accepted by both SQL dialects.
// PostgreSQL timestamp has microsecond precision and accepts offsets below 16h.
const ISODateTimePattern = `^[0-9]{4}-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01])T([01][0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9]([.][0-9]{1,6})?(Z|[+-](0[0-9]|1[0-5]):[0-5][0-9])$`

var isoDateTimePattern = regexp.MustCompile(ISODateTimePattern)

func parseISODateTime(value string) (time.Time, bool) {
	if !isoDateTimePattern.MatchString(value) || value[:4] == "0000" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	return parsed, err == nil
}
