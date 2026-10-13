package cli

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func publishedParamSchema(t *testing.T, kind string) map[string]any {
	t.Helper()
	out, err := captureStdout(t, func() error {
		_, err := executeRootArgs(t, "schema", kind)
		return err
	})
	if err != nil {
		t.Fatalf("onebase schema %s: %v", kind, err)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("onebase schema %s returned invalid JSON: %v", kind, err)
	}
	return schemaAt(t, doc, "properties", "params", "items")
}

func TestSchemaReportParamPeriodEnd(t *testing.T) {
	param := publishedParamSchema(t, "report")
	if param["additionalProperties"] != false {
		t.Fatalf("report parameters must reject unknown keys: %#v", param["additionalProperties"])
	}
	if !reflect.DeepEqual(param["required"], []any{"name", "type"}) {
		t.Errorf("required = %#v, want name and type only", param["required"])
	}
	props := schemaAt(t, param, "properties")
	flag := schemaAt(t, props, "period_end")
	if flag["type"] != "boolean" {
		t.Errorf("period_end.type = %#v, want boolean", flag["type"])
	}
	if description, _ := flag["description"].(string); !strings.Contains(description, "date") {
		t.Errorf("period_end description must explain its date-only scope: %q", description)
	}
	// The aggregate command must publish the same contract as schema report.
	aggregate := schemaAt(t, publishedSchema(t), "$defs", "report", "properties", "params", "items")
	if !reflect.DeepEqual(param, aggregate) {
		t.Error("schema report and schema.$defs.report publish different parameter contracts")
	}
	processor := publishedParamSchema(t, "processor")
	if _, ok := schemaAt(t, processor, "properties")["period_end"]; ok {
		t.Error("period_end is supported only by reports, not processor parameters")
	}
	if processor["additionalProperties"] != false {
		t.Error("processor parameter schema must remain closed")
	}
}

func TestSchemaReportParamCoversModelKeys(t *testing.T) {
	keys := yamlTagsOfStruct(t, "../report/report.go", "Param")
	props := schemaAt(t, publishedParamSchema(t, "report"), "properties")
	// These are pre-existing gaps in the published schema, outside #2119.
	// Keep them explicit so a new model key cannot silently go missing.
	existingGaps := map[string]string{
		"labels":  "Localized labels were already missing before period_end was added",
		"default": "Parameter defaults were already missing before period_end was added",
	}
	known := make(map[string]bool, len(keys))
	for _, key := range keys {
		known[key] = true
		if _, ok := props[key]; !ok && existingGaps[key] == "" {
			t.Errorf("report parameter schema does not describe YAML key %q", key)
		}
	}
	for key := range props {
		if !known[key] {
			t.Errorf("schema promises key %q that report.Param does not read", key)
		}
	}
	for key, reason := range existingGaps {
		if !known[key] || strings.TrimSpace(reason) == "" {
			t.Errorf("invalid existing schema gap %q: %q", key, reason)
		}
		if _, ok := props[key]; ok {
			t.Errorf("key %q is now published; remove its existing-gap exemption", key)
		}
	}
}
