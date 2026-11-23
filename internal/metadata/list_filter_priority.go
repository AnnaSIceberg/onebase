package metadata

// ListFilterAccountPriority orders the choices of one reference filter in
// selected document lists. The target catalog links to a platform account.
type ListFilterAccountPriority struct {
	Documents    []string                    `yaml:"documents"`
	Field        string                      `yaml:"field"`
	AccountField string                      `yaml:"account_field"`
	NameField    string                      `yaml:"name_field"`
	Roles        []string                    `yaml:"roles"`
	Staffing     *ListFilterStaffingPriority `yaml:"staffing,omitempty"`
}

// ListFilterStaffingPriority classifies catalog entries by their latest staff assignment.
type ListFilterStaffingPriority struct {
	Register        string `yaml:"register"`
	DepartmentField string `yaml:"department_field"`
	PositionField   string `yaml:"position_field"`
	ArchiveField    string `yaml:"archive_field,omitempty"`
	Department      string `yaml:"department"`
	Position        string `yaml:"position"`
}
