package report

import "fmt"

// Match the existing saved-settings budget; execution must not silently fall
// back to an unfiltered report when the explicitly supplied JSON is invalid.
const MaxSourceSettingsBytes = 64 * 1024

func ParseSourceSettings(raw string) (*UserReportSettings, error) {
	if len(raw) > MaxSourceSettingsBytes {
		return nil, fmt.Errorf("report source settings exceed 64 KiB")
	}
	settings, err := ParseUserSettings(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid report source settings: %w", err)
	}
	return settings, nil
}
