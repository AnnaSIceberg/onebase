package metadata

import (
	"strings"
	"testing"
)

func TestInfoRegListRejectsUnknownOrDuplicateColumns(t *testing.T) {
	for _, tc := range []struct{ name, yaml, want string }{
		{"unknown", infoRegIndexHead + "list: {columns: [{field: Чужое}]}", "не найдено"},
		{"duplicate", infoRegIndexHead + "list: {columns: [{field: Адрес}, {field: адрес}]}", "дважды"},
		{"no-period", strings.Replace(infoRegIndexHead, "periodic: true", "periodic: false", 1) + "list: {columns: [{field: period}]}", "не найдено"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := loadInfoRegBody(t, tc.yaml); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("LoadInfoRegisterFile error = %v, want %q", err, tc.want)
			}
		})
	}
}
