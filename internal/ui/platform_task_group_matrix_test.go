package ui

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dbtest"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/project"
	"github.com/ivantit66/onebase/internal/runtime"
	"github.com/ivantit66/onebase/internal/storage"
	"github.com/ivantit66/onebase/internal/widget"
	"gopkg.in/yaml.v3"
)

// Private diagnostic only. Exact configuration query through public widget
// runner; every dialect uses new disposable dbtest data, never the DEV copy.
func TestPlatformTaskGroupMatrix(t *testing.T) {
	stage, candidatePath := os.Getenv("PLATFORM_MATRIX_PROJECT"), os.Getenv("PLATFORM_TASK_WIDGET_CANDIDATE")
	if stage == "" || candidatePath == "" {
		t.Skip("explicit configuration fixture required")
	}
	raw, err := os.ReadFile(candidatePath)
	if err != nil {
		t.Fatal(err)
	}
	var candidate metadata.Widget
	if err = yaml.Unmarshal(raw, &candidate); err != nil {
		t.Fatal(err)
	}
	// metadata.Widget uses the loader's snake-case keys; load the original for
	// columns/source/styles below, use the exact candidate query from YAML.
	var payload struct {
		Query string `yaml:"query"`
	}
	if err = yaml.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(payload.Query, "СГРУППИРОВАТЬ ПО 1,2,3,4,5,6,7,8,9,10,11,12") || !strings.Contains(payload.Query, "З.Дата УБЫВ, З.Номер УБЫВ, З.Ссылка УБЫВ") {
		t.Fatal("candidate shape differs")
	}
	proj, err := project.Load(stage)
	if err != nil {
		t.Fatal(err)
	}
	defer proj.Close()
	var original *metadata.Widget
	for _, w := range proj.Widgets {
		if w.Name == "СписокЗадач" {
			original = w
		}
	}
	if original == nil {
		t.Fatal("baseline widget missing")
	}
	stableOriginal := strings.Replace(original.Query, "УПОРЯДОЧИТЬ ПО З.Дата УБЫВ", "УПОРЯДОЧИТЬ ПО З.Дата УБЫВ, З.Номер УБЫВ, З.Ссылка УБЫВ", 1)
	expected := strings.Replace(stableOriginal, "ВЫБРАТЬ РАЗЛИЧНЫЕ", "ВЫБРАТЬ", 1)
	expected = strings.Replace(expected, "УПОРЯДОЧИТЬ ПО", "СГРУППИРОВАТЬ ПО 1,2,3,4,5,6,7,8,9,10,11,12\nУПОРЯДОЧИТЬ ПО", 1)
	normalize := func(s string) string { return strings.Join(strings.Fields(s), " ") }
	if normalize(expected) != normalize(payload.Query) {
		t.Fatal("candidate has extra query changes")
	}
	roles, err := auth.LoadRolesYAML(filepath.Join(stage, "roles"))
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(raw)
	t.Logf("CANDIDATE_SHA256 %s", hex.EncodeToString(digest[:]))
	dbtest.ForEachDialect(t, func(t *testing.T, db *storage.DB) {
		ctx := context.Background()
		wanted := map[string]string{
			"А_Задача":  "Номер Дата Наименование ТипЗадачи Инициатор Исполнитель ФилиалАйсберг Выполнена Примечание ПризнакКлиентПыталсяДозвониться Реклама Подразделение Направление СрокПродления",
			"А_Реклама": "Наименование ЦветЗадачиВСписке", "МРМСписокЗаявокЦветФона": "Наименование Цвет",
			"Пользователи": "Наименование УчётнаяЗапись Подразделение Должность Архив", "ДолжностиОрганизаций": "Наименование", "Подразделения": "Код Наименование", "А_ТипЗадач": "Код Наименование", "Айсберг_Филиалы": "Наименование", "Айсберг_Направления": "Наименование",
		}
		entities := []*metadata.Entity{}
		byName := map[string]*metadata.Entity{}
		for _, e := range proj.Entities {
			if names, ok := wanted[e.Name]; ok {
				copy := &metadata.Entity{Name: e.Name, Kind: e.Kind, Hierarchical: e.Hierarchical}
				for _, f := range e.Fields {
					if strings.Contains(" "+names+" ", " "+f.Name+" ") {
						copy.Fields = append(copy.Fields, f)
					}
				}
				if e.Name == "Пользователи" {
					for _, tp := range e.TableParts {
						if tp.Name == "СписокНаправлений" {
							copy.TableParts = []metadata.TablePart{tp}
						}
					}
				}
				entities = append(entities, copy)
				byName[e.Name] = copy
			}
		}
		if len(entities) != len(wanted) {
			t.Fatal("fixture metadata missing")
		}
		var ir *metadata.InfoRegister
		for _, r := range proj.InfoRegisters {
			if r.Name == "А_ДолжностиПодразделения" {
				ir = r
			}
		}
		if ir == nil {
			t.Fatal("job register missing")
		}
		repo := auth.NewRepo(db)
		if err := repo.EnsureSchema(ctx); err != nil {
			t.Fatal(err)
		}
		if err := db.Migrate(ctx, entities); err != nil {
			t.Fatal(err)
		}
		if err := db.MigrateInfoRegisters(ctx, []*metadata.InfoRegister{ir}); err != nil {
			t.Fatal(err)
		}
		reg := runtime.NewRegistry()
		reg.Load(runtime.LoadOptions{Entities: entities, InfoRegs: []*metadata.InfoRegister{ir}})
		put := func(name string, data map[string]any) uuid.UUID {
			id := uuid.New()
			if err := db.Upsert(ctx, name, id, data, byName[name]); err != nil {
				t.Fatal(err)
			}
			return id
		}
		dept := put("Подразделения", map[string]any{"Код": "000000003", "Наименование": "Synthetic KC"})
		child := put("Подразделения", map[string]any{"Код": "child", "Наименование": "Synthetic Child", "parent_id": dept})
		otherDept := put("Подразделения", map[string]any{"Код": "other", "Наименование": "Synthetic Other"})
		pos := put("ДолжностиОрганизаций", map[string]any{"Наименование": "Сотрудник"})
		managerPos := put("ДолжностиОрганизаций", map[string]any{"Наименование": "Менеджер направления"})
		direction := put("Айсберг_Направления", map[string]any{"Наименование": "Synthetic direction"})
		otherDir := put("Айсберг_Направления", map[string]any{"Наименование": "Synthetic other"})
		filial := put("Айсберг_Филиалы", map[string]any{"Наименование": "Synthetic A"})
		filialB := put("Айсберг_Филиалы", map[string]any{"Наименование": "Synthetic B"})
		color := put("МРМСписокЗаявокЦветФона", map[string]any{"Наименование": "Synthetic color", "Цвет": "#aabbcc"})
		ad := put("А_Реклама", map[string]any{"Наименование": "Synthetic ad", "ЦветЗадачиВСписке": color})
		typ := put("А_ТипЗадач", map[string]any{"Наименование": "Synthetic type", "Код": "000000037"})
		now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
		date := now.Add(-time.Hour)
		tomorrow := time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)
		// First auth account is admin only to satisfy the repository invariant.
		if _, err := repo.Create(ctx, "matrix-bootstrap", uuid.NewString()+"Aa9!", "synthetic", true); err != nil {
			t.Fatal(err)
		}
		type actor struct {
			name  string
			user  *auth.User
			staff uuid.UUID
		}
		actors := []actor{}
		for _, spec := range []struct {
			name           string
			roles          []string
			staff, manager bool
		}{
			{"KC", []string{"ОператорКЦ"}, true, false}, {"pair", []string{"ОператорКЦ", "ОператорМессенджера"}, true, false}, {"chief", []string{"РуководительКЦ"}, true, false}, {"OTK", []string{"ОТКСотрудник"}, true, false}, {"manager", []string{"МенеджерНаправления"}, true, true}, {"NoStaff", []string{"ОператорКЦ", "ОператорМессенджера"}, false, false}, {"ownStaff", []string{"ОператорКЦ", "ОператорМессенджера"}, true, false},
		} {
			u, err := repo.Create(ctx, "matrix-"+spec.name, uuid.NewString()+"Aa9!", "synthetic", false)
			if err != nil {
				t.Fatal(err)
			}
			for _, name := range spec.roles {
				found := false
				for _, role := range roles {
					if role.Name == name {
						u.Roles = append(u.Roles, role)
						found = true
					}
				}
				if !found {
					t.Fatalf("role missing %s", name)
				}
			}
			a := actor{name: spec.name, user: u}
			if spec.staff {
				position := pos
				if spec.manager {
					position = managerPos
				}
				a.staff = put("Пользователи", map[string]any{"Наименование": "synthetic " + spec.name, "УчётнаяЗапись": uuid.MustParse(u.ID), "Подразделение": dept, "Должность": position, "Архив": false})
				// Two matching historical rows exercise duplicate JOIN expansion. Include
				// a future row; query must retain its existing raw-register semantics.
				for _, period := range []time.Time{now.AddDate(-2, 0, 0), now.AddDate(-1, 0, 0), now.AddDate(1, 0, 0)} {
					p := period
					if err := db.InfoRegSet(ctx, ir, map[string]any{"Пользователь": a.staff}, map[string]any{"Подразделение": dept, "Должность": position}, &p); err != nil {
						t.Fatal(err)
					}
				}
				tp := byName["Пользователи"].TableParts[0]
				if err := db.UpsertTablePartRows(ctx, "Пользователи", tp.Name, a.staff, []map[string]any{{"Направление": direction}, {"Направление": direction}, {"Направление": nil}}, tp); err != nil {
					t.Fatal(err)
				}
			}
			actors = append(actors, a)
		}
		// >20 same-date and same-number rows force unique third-key ordering.
		for i := 0; i < 48; i++ {
			data := map[string]any{"Номер": "same-number", "Дата": date, "Наименование": fmt.Sprintf("synthetic-%02d", i), "Подразделение": dept, "Направление": direction, "Выполнена": false, "ФилиалАйсберг": filial, "Примечание": nil, "ПризнакКлиентПыталсяДозвониться": nil}
			if i%2 == 0 {
				data["ТипЗадачи"] = typ
				data["Реклама"] = ad
				data["ПризнакКлиентПыталсяДозвониться"] = true
			}
			if i%3 == 0 {
				data["ФилиалАйсберг"] = filialB
			}
			put("А_Задача", data)
		}
		for _, a := range actors {
			if a.staff != uuid.Nil {
				put("А_Задача", map[string]any{"Номер": "own", "Дата": date, "Наименование": "synthetic assigned", "Подразделение": dept, "Направление": direction, "Исполнитель": a.staff, "Инициатор": a.staff, "Выполнена": false})
			}
		}
		for _, patch := range []map[string]any{{"Дата": now.AddDate(0, 0, 2)}, {"СрокПродления": now.Add(time.Hour)}, {"СрокПродления": now.Add(-time.Hour)}, {"Выполнена": true}, {"Выполнена": nil}, {"Направление": nil}, {"Направление": otherDir}, {"Подразделение": nil}, {"Подразделение": child}, {"Подразделение": otherDept}, {"ФилиалАйсберг": nil}, {"Наименование": nil, "Примечание": nil, "Реклама": nil, "ТипЗадачи": nil, "Инициатор": nil, "ПризнакКлиентПыталсяДозвониться": nil}} {
			data := map[string]any{"Номер": "edge", "Дата": date, "Наименование": "synthetic edge", "Подразделение": dept, "Направление": direction, "Выполнена": false, "ФилиалАйсберг": filial}
			for k, v := range patch {
				data[k] = v
			}
			put("А_Задача", data)
		}
		for _, name := range []string{"Выполнена", "ПризнакКлиентПыталсяДозвониться", "Примечание", "ТипЗадачи", "ФилиалАйсберг"} {
			column := ""
			for _, field := range byName["А_Задача"].Fields {
				if field.Name == name {
					column = metadata.ColumnName(field)
				}
			}
			if column == "" {
				t.Fatal("null fixture field missing")
			}
			var count int
			if err := db.QueryRow(ctx, "SELECT COUNT(*) FROM "+metadata.TableName("А_Задача")+" WHERE "+column+" IS NULL").Scan(&count); err != nil || count < 1 {
				t.Fatalf("NULL fixture missing for %s", name)
			}
			t.Logf("NULL_FIXTURE %s count=%d", name, count)
		}
		canonical := func(rows []map[string]any) []byte {
			all := []string{}
			for _, row := range rows {
				b, _ := json.Marshal(row)
				all = append(all, string(b))
			}
			sort.Strings(all)
			out, _ := json.Marshal(all)
			return out
		}
		for _, a := range actors {
			t.Run(a.name, func(t *testing.T) {
				runner := widget.New(reg, db)
				runner.User = a.user
				runner.CurrentUser = a.user.Login
				for _, allDates := range []bool{false, true} {
					for _, branch := range []any{nil, filial.String(), filialB.String()} {
						t.Run(fmt.Sprintf("all=%t/branch=%v", allDates, branch != nil), func(t *testing.T) {
							params := map[string]any{"ВсеДаты": allDates, "Филиал": branch, "Сейчас": now, "Завтра": tomorrow}
							run := func(text string, limit int) widget.Result {
								w := *original
								w.Query = text
								w.Limit = limit
								return runner.RunWithOptions(ctx, &w, widget.RunOptions{Params: params, Fresh: true})
							}
							before, after := run(stableOriginal, 0), run(payload.Query, 0)
							if before.Error != "" || after.Error != "" {
								t.Fatalf("query error old=%s new=%s", before.Error, after.Error)
							}
							if before.AccessDenied != after.AccessDenied {
								t.Fatal("access changed")
							}
							if before.AccessDenied {
								t.Fatal("non-vacuous role fixture unexpectedly denied")
							}
							bc, _ := json.Marshal(before.Columns)
							ac, _ := json.Marshal(after.Columns)
							if !bytes.Equal(canonical(before.Rows), canonical(after.Rows)) || !bytes.Equal(bc, ac) {
								t.Fatal("complete values/columns/styles/URLs changed")
							}
							orderedB, _ := json.Marshal(before.Rows)
							orderedA, _ := json.Marshal(after.Rows)
							if !bytes.Equal(orderedB, orderedA) {
								t.Fatal("same three keys yielded different complete ordering")
							}
							b20, a20, repeat := run(stableOriginal, 20), run(payload.Query, 20), run(payload.Query, 20)
							x, _ := json.Marshal(b20.Rows)
							y, _ := json.Marshal(a20.Rows)
							z, _ := json.Marshal(repeat.Rows)
							if b20.Error != "" || a20.Error != "" || repeat.Error != "" || !bytes.Equal(x, y) || !bytes.Equal(y, z) {
								t.Fatal("deterministic top20/limit differs")
							}
							if branch == nil && !allDates && len(after.Rows) <= 20 {
								t.Fatalf("tie fixture too small: %d", len(after.Rows))
							}
							t.Logf("EQUIVALENT full=%d limit=%d", len(after.Rows), len(a20.Rows))
						})
					}
				}
			})
		}
	})
}
