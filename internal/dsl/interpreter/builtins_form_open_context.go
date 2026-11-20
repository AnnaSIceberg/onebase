package interpreter

import "fmt"

func init() {
	unavailable := BuiltinFunc(func(_ []any, _ string, _ int) (any, error) {
		return nil, fmt.Errorf("ОткрытьДокументСКонтекстом доступна в событии формы сохранённого документа")
	})
	builtins["открытьдокументсконтекстом"] = unavailable
	builtins["opendocumentwithcontext"] = unavailable
	empty := BuiltinFunc(func(_ []any, _ string, _ int) (any, error) { return "", nil })
	builtins["контекстоткрытияформы"] = empty
	builtins["formopeningcontext"] = empty
}
