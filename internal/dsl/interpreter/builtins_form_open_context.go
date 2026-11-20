package interpreter

import "fmt"

func init() {
	unavailable := BuiltinFunc(func(_ []any, _ string, _ int) (any, error) {
		return nil, fmt.Errorf("ОткрытьДокументСКонтекстом доступна в событии формы сохранённого документа")
	})
	builtins["открытьдокументсконтекстом"] = unavailable
	builtins["opendocumentwithcontext"] = unavailable
	builtins["показатьссылкунайденнойзаявки"] = func(_ []any, _ string, _ int) (any, error) {
		return nil, fmt.Errorf("ПоказатьСсылкуНайденнойЗаявки доступна в ПослеОткрытия сохранённой формы с link_command")
	}
	builtins["открытьнайденнуюзаявку"] = func(_ []any, _ string, _ int) (any, error) {
		return nil, fmt.Errorf("ОткрытьНайденнуюЗаявку: ссылка отсутствует или истекла")
	}
	empty := BuiltinFunc(func(_ []any, _ string, _ int) (any, error) { return "", nil })
	builtins["контекстоткрытияформы"] = empty
	builtins["formopeningcontext"] = empty
}
