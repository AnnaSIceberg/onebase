package ui

// report_compose_walk.go — единый обход дерева скомпонованного отчёта (СКД).
// HTML-таблица (report_compose_render.go) и Excel-выгрузка (report_compose_excel.go)
// прежде обходили это дерево независимо, дублируя порядок строк и структуру; копии
// уже начали расходиться. Теперь обе строятся поверх walkComposed — порядок задан
// в одном месте, потребитель лишь решает, как нарисовать каждую строку.

import (
	"strconv"

	"github.com/ivantit66/onebase/internal/report"
	"github.com/ivantit66/onebase/internal/report/compose"
)

// composeSink принимает события обхода в порядке рендера:
//
//	группа с итогами → [дочерние группы] → [детали] … → [общий итог]
//
// level — глубина строки: группа на level, её детали на level+1.
// path — экранированный путь группы (для HTML data-group/data-parent; Excel
// его игнорирует).
type composeSink interface {
	group(g *compose.Group, level int, path string)
	detail(d compose.DetailRow, level int, path string)
	grand(grand map[string]any)
}

// walkComposed обходит дерево групп результата компоновки и вызывает методы sink
// в порядке рендера. Детали выводятся только при spec.Detail,
// общий итог — при spec.Totals.Grand. Итоги групп всегда в их заголовках.
func walkComposed(res *compose.Result, spec *report.Composition, s composeSink) {
	var walk func(g *compose.Group, level int, parentPath string, position int)
	walk = func(g *compose.Group, level int, parentPath string, position int) {
		path := parentPath + "/" + reportGroupPathSegment(g.Key, g.ReferenceGroup, position)
		s.group(g, level, path)
		for i, ch := range g.Children {
			walk(ch, level+1, path, i)
		}
		if spec.Detail {
			for _, d := range g.Details {
				s.detail(d, level+1, path)
			}
		}
		// A group's totals already occupy its header row. Do not repeat them
		// as a footer, including legacy totals.subtotals configurations.
	}
	for i, g := range res.Groups {
		walk(g, 0, "", i)
	}
	if spec.Detail {
		for _, d := range res.Details {
			s.detail(d, 0, "")
		}
	}
	if spec.Totals.Grand {
		s.grand(res.Grand)
	}
}

func reportGroupPathSegment(key any, reference bool, position int) string {
	if reference {
		// Ordinary labels escape percent signs in pathSeg. The prefix is
		// reserved for request-local group positions and contains no UUID.
		return "%ref" + strconv.Itoa(position)
	}
	return pathSeg(fmtVal(key))
}
