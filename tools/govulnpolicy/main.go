// Command govulnpolicy — фильтр потока `govulncheck -format json`: проверка
// падает на любой ДОСТИЖИМОЙ (called) уязвимости.
//
// Исключений сейчас нет: любая достижимая уязвимость запрещена.
// GO-2026-6452 исправлена начиная с Excelize 2.11.0 по Go vulnerability database;
// прежнее исключение для точного upstream-коммита больше не требуется.
//
// Использование (см. job vuln в .github/workflows/ci.yml):
//
//	govulncheck -json ./... | go run ./tools/govulnpolicy
//
// Коды выхода: 0 — достижимых advisory нет; 1 — достижимая
// уязвимость или битый поток govulncheck.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// finding/trace — подмножество схемы govulncheck -format json.
type message struct {
	Config  *json.RawMessage `json:"config"`
	Finding *finding         `json:"finding"`
}

type finding struct {
	OSV   string  `json:"osv"`
	Trace []frame `json:"trace"`
}

type frame struct {
	Module   string `json:"module"`
	Version  string `json:"version"`
	Function string `json:"function"`
}

type violation struct {
	osv     string
	version string
	detail  string
}

func main() {
	if err := run(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "govulnpolicy:", err)
		os.Exit(1)
	}
}

func run(r io.Reader, out io.Writer) error {
	dec := json.NewDecoder(r)
	var (
		sawConfig    bool
		calledOSVs   = map[string]string{} // osv → версия модуля из trace[0]
		skippedLevel int                   // module-level находки (не called) — govulncheck их не считает «affected»
		violations   []violation
	)
	for {
		var msg message
		if err := dec.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("битый поток govulncheck (обрыв или не JSON): %v", err)
		}
		if msg.Config != nil {
			sawConfig = true
			continue
		}
		if msg.Finding == nil {
			continue
		}
		f := msg.Finding
		var called bool
		for _, fr := range f.Trace {
			if fr.Function != "" {
				called = true
				break
			}
		}
		if !called {
			skippedLevel++
			continue
		}
		version := ""
		if len(f.Trace) > 0 {
			version = f.Trace[0].Version
		}
		if _, seen := calledOSVs[f.OSV]; !seen {
			calledOSVs[f.OSV] = version
		}
	}

	if !sawConfig {
		return fmt.Errorf("govulncheck не выдал config — запуск оборвался, считать результат нельзя")
	}

	for osv, version := range calledOSVs {
		violations = append(violations, violation{osv: osv, version: version,
			detail: "достижимая уязвимость"})
	}

	for _, v := range violations {
		// stdout/stderr — не вердикт: ошибка записи не меняет исход проверки.
		_, _ = fmt.Fprintf(out, "ЗАПРЕЩЕНО: %s (модуль-версия %s): %s\n", v.osv, v.version, v.detail)
	}
	if len(violations) > 0 {
		return fmt.Errorf("%d достижимых уязвимостей нарушают политику", len(violations))
	}

	_, _ = fmt.Fprintf(out, "govulnpolicy: OK — достижимых уязвимостей нет; module-level пропущено %d.\n", skippedLevel)
	return nil
}
