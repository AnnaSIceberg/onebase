package query

// ProjectionLineage — один элемент списка выборки любого SELECT запроса, включая
// вложенные и ветви ОБЪЕДИНИТЬ. Нужен обязательной политике ПДн отчётов там, где
// поколоночное сопоставление (Projection.Columns) невозможно: по нему видно,
// доходит ли значение защищённого поля до результата — напрямую или через
// алиас подзапроса.
type ProjectionLineage struct {
	// Output — имя колонки, под которым элемент виден внешнему запросу: алиас
	// «КАК …», иначе последний идентификатор простой ссылки на поле. Для
	// выражения без алиаса пусто.
	Output string
	// Uses — идентификаторы, чьё значение может попасть в колонку (поля, алиасы
	// таблиц и колонок подзапросов). Не входят только условия «КОГДА … ТОГДА» и
	// аргументы КОЛИЧЕСТВО(…): они дают признак или счётчик, а не само значение.
	// В остальном избыточность намеренная: лишнее имя только ужесточает.
	Uses []string
	// Plain — элемент является простой ссылкой на поле (маскируется на выходе).
	Plain bool
	// Star — элемент «*» или «Т.*».
	Star bool
	// Result — элемент принадлежит SELECT, чьи строки являются результатом
	// запроса: верхнему уровню или ветви ОБЪЕДИНИТЬ, а не подзапросу в ИЗ/ГДЕ.
	Result bool
}

// projectionLineage разбирает списки выборки всех SELECT потока токенов.
func projectionLineage(tokens []tok) []ProjectionLineage {
	var out []ProjectionLineage
	// resultParen[d] — скобка глубины d открывает ветвь результата
	// («(ВЫБРАТЬ …) ОБЪЕДИНИТЬ (ВЫБРАТЬ …)»), а не подзапрос.
	resultParen := map[int]bool{0: true}
	depth := 0
	for i, t := range tokens {
		switch t.kind {
		case tLParen:
			depth++
			resultParen[depth] = resultParen[depth-1] && opensResultBranch(tokens, i)
			continue
		case tRParen:
			if depth > 0 {
				depth--
			}
			continue
		case tIdent:
		default:
			continue
		}
		if kw, isKW := sqlKW(t.val); !isKW || kw != "SELECT" {
			continue
		}
		start := selectProjectionStart(tokens, i)
		end := projectionEnd(tokens, start)
		result := resultParen[depth]
		for _, item := range splitProjectionItems(tokens[start:end]) {
			col, _ := parseProjectionItem(item)
			line := ProjectionLineage{Uses: valueIdentifiers(item), Result: result, Star: col.Star}
			if !col.Star {
				line.Output = col.Output
				line.Plain = len(col.Fields) > 0
			}
			out = append(out, line)
		}
	}
	return out
}

// opensResultBranch — скобка в позиции i начинает ветвь результата: стоит в
// начале запроса, после другой такой скобки или после ОБЪЕДИНИТЬ [ВСЕ], и сразу
// за ней идёт ВЫБРАТЬ.
func opensResultBranch(tokens []tok, i int) bool {
	if i+1 >= len(tokens) || tokens[i+1].kind != tIdent {
		return false
	}
	if kw, isKW := sqlKW(tokens[i+1].val); !isKW || kw != "SELECT" {
		return false
	}
	if i == 0 {
		return true
	}
	prev := tokens[i-1]
	if prev.kind == tLParen {
		return true
	}
	if prev.kind != tIdent {
		return false
	}
	kw, isKW := sqlKW(prev.val)
	if isKW && kw == "ALL" && i >= 2 && tokens[i-2].kind == tIdent {
		kw, isKW = sqlKW(tokens[i-2].val)
	}
	return isKW && kw == "UNION"
}

// projectionEnd — конец списка выборки, начатого в start: ИЗ, ОБЪЕДИНИТЬ или
// ключевое слово хвоста той же глубины, закрывающая скобка охватывающего
// подзапроса или конец потока.
func projectionEnd(tokens []tok, start int) int {
	depth := 0
	for i := start; i < len(tokens); i++ {
		switch tokens[i].kind {
		case tLParen:
			depth++
		case tRParen:
			if depth == 0 {
				return i
			}
			depth--
		case tIdent:
			if depth != 0 {
				continue
			}
			if kw, isKW := sqlKW(tokens[i].val); isKW {
				switch kw {
				case "FROM", "UNION", "WHERE", "GROUP", "HAVING", "ORDER", "INTO":
					return i
				}
			}
		}
	}
	return len(tokens)
}

// valueIdentifiers — identifiersIn без условий ВЫБОР КОГДА … ТОГДА и без
// аргументов КОЛИЧЕСТВО(…).
func valueIdentifiers(item []tok) []string {
	var out []string
	caseDepth, condDepth := 0, -1
	countParen, depth := -1, 0
	for i, t := range item {
		switch t.kind {
		case tLParen:
			depth++
			continue
		case tRParen:
			if depth == countParen {
				countParen = -1
			}
			if depth > 0 {
				depth--
			}
			continue
		case tIdent:
		default:
			continue
		}
		if kw, isKW := sqlKW(t.val); isKW {
			switch kw {
			case "CASE":
				caseDepth++
			case "END":
				if caseDepth > 0 {
					caseDepth--
				}
			case "WHEN":
				if condDepth < 0 {
					condDepth = caseDepth
				}
			case "THEN":
				if condDepth == caseDepth {
					condDepth = -1
				}
			}
			continue
		}
		if i+1 < len(item) && item[i+1].kind == tLParen {
			if countParen < 0 && isCountFunc(t.val) {
				countParen = depth + 1
			}
			continue // имя функции
		}
		if condDepth >= 0 || countParen >= 0 {
			continue
		}
		out = append(out, t.val)
	}
	return out
}

func isCountFunc(name string) bool {
	switch upperFast(name) {
	case "КОЛИЧЕСТВО", "COUNT":
		return true
	}
	return false
}

// paramPredicateIdentifiers — идентификаторы, сравниваемые в одном условии с
// параметром запроса («Телефон = &Т», «ПОДСТРОКА(Телефон, 1, 3) В (&Список)»).
// Параметр — ввод пользователя, поэтому такое условие по защищённому полю —
// оракул перебора, даже если само поле не выводится. Условие ограничено
// И/ИЛИ/ГДЕ/ПО/ИМЕЯ/КОГДА/ТОГДА/ИНАЧЕ, границами SELECT и запятыми вне
// аргументов функции.
func paramPredicateIdentifiers(tokens []tok) []string {
	set := newFieldSet()
	var atom []string
	hasParam := false
	flush := func() {
		if hasParam {
			set.addAll(atom)
		}
		atom, hasParam = atom[:0], false
	}
	var funcParens []bool
	for i, t := range tokens {
		switch t.kind {
		case tParam:
			hasParam = true
		case tLParen:
			isFunc := false
			if i > 0 && tokens[i-1].kind == tIdent {
				_, isKW := sqlKW(tokens[i-1].val)
				isFunc = !isKW
			}
			funcParens = append(funcParens, isFunc)
		case tRParen:
			if n := len(funcParens); n > 0 {
				funcParens = funcParens[:n-1]
			}
		case tComma:
			if n := len(funcParens); n == 0 || !funcParens[n-1] {
				flush()
			}
		case tIdent:
			if kw, isKW := sqlKW(t.val); isKW {
				switch kw {
				case "AND", "OR", "WHERE", "ON", "HAVING", "CASE", "WHEN", "THEN", "ELSE", "END",
					"SELECT", "FROM", "UNION", "GROUP", "ORDER":
					flush()
				}
				continue
			}
			if i+1 < len(tokens) && tokens[i+1].kind == tLParen {
				continue // имя функции
			}
			atom = append(atom, t.val)
		}
	}
	flush()
	return set.list()
}
