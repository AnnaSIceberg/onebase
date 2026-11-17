package access_test

import (
	"context"
	"testing"

	"github.com/ivantit66/onebase/internal/access"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/query"
)

func TestReportOutputDiscloseAndCardPoliciesStaySeparate(t *testing.T) {
	meta := &metadata.Entity{Name: "Source", Kind: metadata.KindCatalog, Presentation: []string{"Label"}, Fields: []metadata.Field{
		{Name: "Phone", Type: metadata.FieldTypeString, PII: true},
		{Name: "House", Type: metadata.FieldTypeString, ReportPII: true},
		{Name: "Label", Type: metadata.FieldTypeString},
		{Name: "Amount", Type: metadata.FieldTypeNumber},
	}}
	lookup := func(_, _ string) *metadata.Entity { return meta }
	reader := &auth.User{Roles: []*auth.Role{{Permissions: auth.Permission{Catalogs: map[string][]string{meta.Name: {"read", "disclose"}}}}}}
	res, err := query.Compile("ВЫБРАТЬ Phone, House, Amount ИЗ Справочник.Source ГДЕ Phone = &Phone", query.CompileOpts{Entities: []*metadata.Entity{meta}, Params: map[string]any{"Phone": "SYNTHETIC_PRIVATE"}})
	if err != nil {
		t.Fatal(err)
	}
	if ordinary := access.QueryMaskPlanFor(reader, res, lookup); ordinary.Denied != "" {
		t.Fatal("ordinary disclosed-field search changed", ordinary.Denied)
	}
	if report := access.ReportOutputQueryMaskPlan(reader, res, lookup); report.Denied == "" {
		t.Fatal("disclose bypassed mandatory report predicate denial")
	}
	for _, user := range []*auth.User{{IsAdmin: true}, reader} {
		ordinary := access.FieldDecisions(user, "catalog", meta.Name, meta)
		if ordinary["House"].Masked() {
			t.Fatal("report_pii altered ordinary card rights")
		}
		decisions := access.ReportOutputFieldDecisions(user, "catalog", meta.Name, meta)
		for _, field := range []string{"Phone", "House", "Label"} {
			if !decisions[field].Hidden() {
				t.Fatal("report field not hidden", field)
			}
		}
	}
	ordinaryCtx := context.Background()
	if access.IsReportOutput(ordinaryCtx) || !access.IsReportOutput(access.WithReportOutput(ordinaryCtx)) {
		t.Fatal("report policy context leaked into an ordinary context")
	}
}

func TestReportOutputEmptyUnknownSourceFailsClosed(t *testing.T) {
	result := query.Result{Sources: []query.SourceRef{{}}}
	if access.ReportOutputQueryMaskPlan(&auth.User{IsAdmin: true}, result, nil).Denied == "" || access.QueryMaskPlanFor(nil, result, nil).Denied == "" {
		t.Fatal("empty source name turned unresolved metadata into an allowed plan")
	}
}

func TestReportOutputPolicyMandatoryForEveryRole(t *testing.T) {
	meta := &metadata.Entity{Name: "Protected", Kind: metadata.KindCatalog, Presentation: []string{"Label"}, Fields: []metadata.Field{
		{Name: "Phone", Type: metadata.FieldTypeString, PII: true}, {Name: "Label", Type: metadata.FieldTypeString}, {Name: "Amount", Type: metadata.FieldTypeNumber},
	}}
	lookup := func(_, _ string) *metadata.Entity { return meta }
	for _, user := range []*auth.User{nil, {IsAdmin: true}, {Roles: []*auth.Role{{Permissions: auth.Permission{
		Catalogs: map[string][]string{meta.Name: {"read", "disclose"}}, FieldAccess: auth.FieldAccess{Catalogs: map[string]auth.FieldPolicies{meta.Name: {"Phone": {Read: "full"}}}},
	}}}}} {
		compile := func(text string) query.Result {
			result, err := query.Compile(text, query.CompileOpts{Entities: []*metadata.Entity{meta}})
			if err != nil {
				t.Fatal(err)
			}
			return result
		}
		result := compile("ВЫБРАТЬ Phone КАК Contact, Label КАК Title, Amount ИЗ Справочник.Protected")
		rows := []map[string]any{{"contact": "79990009987", "title": "HOUSE_987 FLAT_654", "amount": 12345}}
		plan := access.ReportOutputQueryMaskPlan(user, result, lookup)
		if err := plan.Apply(rows); err != nil {
			t.Fatal(err)
		}
		if rows[0]["contact"] != nil || rows[0]["title"] != nil || rows[0]["amount"] != 12345 {
			t.Fatalf("output: %#v", rows)
		}
		for _, text := range []string{
			"ВЫБРАТЬ Amount ИЗ Справочник.Protected ГДЕ Phone = &Secret",
			"ВЫБРАТЬ Label ИЗ Справочник.Protected УПОРЯДОЧИТЬ ПО Label",
			"ВЫБРАТЬ МАКСИМУМ(Phone) КАК V ИЗ Справочник.Protected",
			"ВЫБРАТЬ Amount ИЗ Справочник.Protected ОБЪЕДИНИТЬ ВСЕ ВЫБРАТЬ Amount ИЗ Справочник.Protected",
		} {
			if plan := access.ReportOutputQueryMaskPlan(user, compile(text), lookup); plan.Denied == "" {
				t.Fatalf("unsafe query accepted: %s", text)
			}
		}
	}
	if access.ReportOutputFilterAllowed(meta, "Phone") || access.ReportOutputFilterAllowed(meta, "LABEL") || access.ReportOutputFilterAllowed(meta, "Unknown") || !access.ReportOutputFilterAllowed(meta, "Amount") || !access.ReportOutputFilterAllowed(meta, "id") {
		t.Fatal("source-field filter gate")
	}
	// The report gate must never change the ordinary card/admin query policy.
	result, err := query.Compile("ВЫБРАТЬ Phone ИЗ Справочник.Protected", query.CompileOpts{Entities: []*metadata.Entity{meta}})
	if err != nil {
		t.Fatal(err)
	}
	rows := []map[string]any{{"phone": "79990009987"}}
	if err := access.QueryMaskPlanFor(&auth.User{IsAdmin: true}, result, lookup).Apply(rows); err != nil {
		t.Fatal(err)
	}
	if rows[0]["phone"] != "79990009987" {
		t.Fatal("ordinary admin policy changed")
	}
}

func TestReportOutputPolicyUnknownSourceFailsClosed(t *testing.T) {
	res := query.Result{Sources: []query.SourceRef{{Kind: "catalog", Name: "Missing"}}}
	if plan := access.ReportOutputQueryMaskPlan(&auth.User{IsAdmin: true}, res, nil); plan.Denied == "" {
		t.Fatal("unresolved source silently removed mandatory report protection")
	}
}

func TestOrdinaryMaskUnknownSourceFailsClosedWithoutChangingKnownSource(t *testing.T) {
	meta := &metadata.Entity{Name: "Known", Kind: metadata.KindCatalog, Fields: []metadata.Field{{Name: "Phone", Type: metadata.FieldTypeString, PII: true}}}
	compiled, err := query.Compile("ВЫБРАТЬ Phone ИЗ Справочник.Known", query.CompileOpts{Entities: []*metadata.Entity{meta}})
	if err != nil {
		t.Fatal(err)
	}
	reader := &auth.User{Roles: []*auth.Role{{Permissions: auth.Permission{Catalogs: map[string][]string{meta.Name: {"read"}}}}}}
	for _, user := range []*auth.User{nil, {IsAdmin: true}, reader} {
		if plan := access.QueryMaskPlanFor(user, compiled, nil); plan.Denied == "" {
			t.Fatal("unresolved ordinary source silently removed PII policy")
		}
		rows := []map[string]any{{"phone": "79990009987"}}
		plan := access.QueryMaskPlanFor(user, compiled, func(_, _ string) *metadata.Entity { return meta })
		if err := plan.Apply(rows); err != nil {
			t.Fatal(err)
		}
		if user == reader {
			if rows[0]["phone"] == "79990009987" {
				t.Fatal("known-source reader lost default mask")
			}
		} else if rows[0]["phone"] != "79990009987" {
			t.Fatal("ordinary known-source admin/nil privileges changed")
		}
	}
}
