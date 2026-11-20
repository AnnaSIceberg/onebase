package storage

import "strings"

const phoneSeparators = " ()-+."

// PhoneLast10SQL uses the same immutable expression in index definitions and
// query predicates. column is a resolved SQL column, optionally qualified.
func PhoneLast10SQL(column string) string {
	digits := column
	for _, separator := range phoneSeparators {
		digits = "REPLACE(" + digits + ", '" + string(separator) + "', '')"
	}
	nonDigits := digits
	for _, digit := range "0123456789" {
		nonDigits = "REPLACE(" + nonDigits + ", '" + string(digit) + "', '')"
	}
	length := "LENGTH(" + digits + ")"
	return "CASE WHEN " + nonDigits + " = '' AND (" + length + " = 10 OR (" +
		length + " = 11 AND SUBSTR(" + digits + ", 1, 1) IN ('7', '8'))) THEN SUBSTR(" +
		digits + ", " + length + " - 9, 10) END"
}

// NormalizePhoneLast10 applies the index key rules to an already-loaded record.
// An empty key cannot participate in a match.
func NormalizePhoneLast10(value string) string {
	var digits [11]byte
	n := 0
	for i := 0; i < len(value); i++ {
		ch := value[i]
		if strings.IndexByte(phoneSeparators, ch) >= 0 {
			continue
		}
		if ch < '0' || ch > '9' || n == len(digits) {
			return ""
		}
		digits[n] = ch
		n++
	}
	if n == 10 {
		return string(digits[:10])
	}
	if n == 11 && (digits[0] == '7' || digits[0] == '8') {
		return string(digits[1:])
	}
	return ""
}
