package compose

import (
	"strings"

	"github.com/google/uuid"
)

const referenceGroupsKey = "\x00onebase.reference_groups"

type referenceGroupID uuid.UUID

type referenceGroup struct {
	id    referenceGroupID
	label string
}

type referenceGroups map[string]referenceGroup

// SetReferenceGroup keeps grouping identity separate from the authorized label.
// Only the authorized caption is copied into detail presentation.
func SetReferenceGroup(row Row, field string, id uuid.UUID, label string) {
	groups, _ := row[referenceGroupsKey].(referenceGroups)
	if groups == nil {
		groups = make(referenceGroups)
		row[referenceGroupsKey] = groups
	}
	groups[strings.ToLower(field)] = referenceGroup{id: referenceGroupID(id), label: label}
}

// ReferenceLabel returns the visible caption while Values retains drilldown input.
func (d DetailRow) ReferenceLabel(field string) (string, bool) {
	label, ok := d.referenceLabels[strings.ToLower(field)]
	return label, ok
}

func groupingKey(row Row, field string, value any) any {
	groups, _ := row[referenceGroupsKey].(referenceGroups)
	if value, ok := groups[strings.ToLower(field)]; ok {
		return value.id
	}
	return normalizeGroupKey(value)
}

func groupingDisplay(row Row, field string, key any) (any, bool) {
	if _, ok := key.(referenceGroupID); ok {
		groups, _ := row[referenceGroupsKey].(referenceGroups)
		return groups[strings.ToLower(field)].label, true
	}
	return key, false
}
