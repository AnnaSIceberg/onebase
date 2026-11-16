package query_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/ivantit66/onebase/internal/dsl/ast"
	"github.com/ivantit66/onebase/internal/dsl/lexer"
	"github.com/ivantit66/onebase/internal/dsl/parser"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/query"
	"github.com/ivantit66/onebase/internal/report"
	"github.com/ivantit66/onebase/internal/storage"
)

// Local compile-only inventory: no connection, SQL execution, module execution,
// browser, deployment or full project validation. Not a native acceptance test.
func TestPlatform3ActualLineageCompileInventory(t *testing.T) {
	root := os.Getenv("PLATFORM3_ACTUAL_ROOT")
	output := os.Getenv("PLATFORM3_ACTUAL_OUT")
	if root == "" || output == "" {
		t.Skip("explicit local metadata paths required")
	}
	opts := query.CompileOpts{Dialect: storage.SQLiteDialect{}}
	for _, kind := range []metadata.Kind{metadata.KindCatalog, metadata.KindDocument} {
		dir := "catalogs"
		if kind == metadata.KindDocument {
			dir = "documents"
		}
		files, err := os.ReadDir(filepath.Join(root, dir))
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range files {
			if file.IsDir() || !strings.HasSuffix(file.Name(), ".yaml") {
				continue
			}
			e, err := metadata.LoadFile(filepath.Join(root, dir, file.Name()), kind)
			if err != nil {
				t.Fatal(err)
			}
			opts.Entities = append(opts.Entities, e)
		}
	}
	for _, item := range []struct {
		dir  string
		kind int
	}{{"registers", 0}, {"inforegs", 1}, {"accountregs", 2}} {
		files, err := os.ReadDir(filepath.Join(root, item.dir))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range files {
			if file.IsDir() || !strings.HasSuffix(file.Name(), ".yaml") {
				continue
			}
			path := filepath.Join(root, item.dir, file.Name())
			switch item.kind {
			case 0:
				r, err := metadata.LoadRegisterFile(path)
				if err != nil {
					t.Fatal(err)
				}
				opts.Registers = append(opts.Registers, r)
			case 1:
				r, err := metadata.LoadInfoRegisterFile(path)
				if err != nil {
					t.Fatal(err)
				}
				opts.InfoRegs = append(opts.InfoRegs, r)
			case 2:
				r, err := metadata.LoadAccountRegisterFile(path)
				if err != nil {
					t.Fatal(err)
				}
				opts.AccountRegs = append(opts.AccountRegs, r)
			}
		}
	}
	type result struct {
		Name, QuerySHA256, FileSHA256, Status, Reason string
		Outputs                                       int
	}
	var results []result
	compile := func(name, text, filehash string) {
		sum := sha256.Sum256([]byte(text))
		entry := result{Name: name, QuerySHA256: hex.EncodeToString(sum[:]), FileSHA256: filehash}
		params := map[string]any{}
		for _, m := range regexp.MustCompile(`&([\pL_][\pL\pN_]*)`).FindAllStringSubmatch(text, -1) {
			params[m[1]] = nil
		}
		local := opts
		local.Params = params
		_, proof, err := query.CompileReportLineage(text, local)
		if err != nil {
			entry.Status = "STOP"
			entry.Reason = err.Error()
		} else {
			entry.Status = "PROVENANCE_ONLY"
			entry.Outputs = len(proof.Outputs())
		}
		results = append(results, entry)
	}
	for _, name := range []string{"А_ЗаявкиПоНомеруТелефона", "А_ОтзывыКлиентовИКоличествоСМС", "А_ЗвонкиПоНаправлениям", "А_ВедомостьПоАкцептованнымОтчетамВендора", "А_ВедомостьПоГарантийнымВРаботе"} {
		path := filepath.Join(root, "reports", name+".yaml")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		r, err := report.LoadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if name == "А_ЗаявкиПоНомеруТелефона" {
			roots := map[string]string{"З1": "Документ.А_Заявка", "З2": "Документ.А_Заявка", "Ф1": "Справочник.Айсберг_Филиалы", "ФН": "Справочник.Айсберг_Филиалы", "Н1": "Справочник.Айсберг_Направления"}
			seen := map[string]bool{}
			for _, field := range regexp.MustCompile(`[\pL_][\pL\pN_]*(?:\.[\pL_][\pL\pN_]*)+`).FindAllString(r.Query, -1) {
				parts := strings.Split(field, ".")
				if seen[field] || roots[parts[0]] == "" { continue }
				seen[field] = true
				_, _, fieldErr := query.CompileReportLineage("ВЫБРАТЬ "+field+" КАК Probe ИЗ "+roots[parts[0]]+" КАК "+parts[0], opts)
				if fieldErr != nil { t.Logf("schema-only path STOP %s: %v", field, fieldErr) }
			}
		}
		sum := sha256.Sum256(data)
		compile(name, r.Query, hex.EncodeToString(sum[:]))
		again, err := os.ReadFile(path)
		if err != nil || sha256.Sum256(again) != sum {
			t.Fatal("input changed during compile inventory")
		}
	}
	path := filepath.Join(root, "src", "а_kpi_операторов_финочтет.rep.os")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	program, err := parser.New(lexer.New(string(data), "<actual-kpi-query>")).ParseProgram()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, proc := range program.Procedures {
		if !strings.EqualFold(proc.Name.Literal, "Выборка") {
			continue
		}
		for _, stmt := range proc.Body {
			if ret, ok := stmt.(*ast.ReturnStmt); ok {
				if literal, ok := ret.Value.(*ast.StringLit); ok {
					sum := sha256.Sum256(data)
					compile("KPI/static-query", literal.Value, hex.EncodeToString(sum[:]))
					found = true
				}
			}
		}
	}
	if !found {
		t.Fatal("KPI static query missing; do not execute module")
	}
	encoded, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(output, encoded, 0600); err != nil {
		t.Fatal(err)
	}
	for _, entry := range results {
		t.Logf("%s: %s (%s)", entry.Name, entry.Status, entry.Reason)
	}
}
