package ui

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"testing"

	"github.com/ivantit66/onebase/internal/metadata"
)

// Реквизит формы без своего контрола — флаг для hidden_when. Сервер реквизиты
// формы между событиями не хранит, поэтому флаг, выставленный обработчиком,
// должен вернуться с клиента следующим событием (скрытое поле managed.js), иначе
// условие откатывается при первом же постороннем событии.
func carrierEventServer(t *testing.T) (*Server, *metadata.Entity) {
	t.Helper()
	s, ent := setupManagedEventsServer(t, `
Процедура ПолеНаимПриИзменении()
	Объект.СкрытьПоле = Истина;
КонецПроцедуры

Процедура ПолеПримечаниеПриИзменении()
	Сообщить("флаг не трогаю");
КонецПроцедуры
`, nil, []*metadata.FormElement{
		{Kind: metadata.FormElementField, Name: "ПолеНаим", DataPath: "Объект.Наименование",
			Handlers: map[metadata.FormEventType]string{metadata.FormEventOnChange: "ПолеНаимПриИзменении"}},
		{Kind: metadata.FormElementField, Name: "ПолеПримечание", DataPath: "Примечание",
			Handlers: map[metadata.FormEventType]string{metadata.FormEventOnChange: "ПолеПримечаниеПриИзменении"}},
		{Kind: metadata.FormElementField, Name: "ПолеСкрываемое", DataPath: "Примечание2", HiddenWhen: "СкрытьПоле"},
	})
	ent.Forms[0].Attributes = []*metadata.FormAttribute{
		{Name: "СкрытьПоле", TypeRef: "bool", Save: false},
		{Name: "Примечание", TypeRef: "string(100)", Save: false},
		{Name: "Примечание2", TypeRef: "string(100)", Save: false},
	}
	return s, ent
}

func TestFormAttrCarrierKeepsHiddenWhenAcrossEvents(t *testing.T) {
	s, ent := carrierEventServer(t)

	first := decodeFormEventResponse(t, executeFormEvent(t, s, ent, url.Values{
		"_element": {"ПолеНаим"}, "_event": {"ПриИзменении"}, "Наименование": {"Тест"},
	}).Body.Bytes())
	if first.ElementStates == nil || !first.ElementStates.Hidden["ПолеСкрываемое"] {
		t.Fatalf("обработчик выставил флаг, а поле не скрыто: %+v", first.ElementStates)
	}
	carried, ok := first.Values["СкрытьПоле"]
	if !ok {
		t.Fatalf("флаг не вернулся в values — клиенту нечего положить в скрытое поле (ключи: %v)", keysOfAny(first.Values))
	}

	// Следующее событие — постороннее, флаг не трогает. Клиент присылает флаг
	// так же, как applyValues записал его в скрытое поле: строкой.
	second := decodeFormEventResponse(t, executeFormEvent(t, s, ent, url.Values{
		"_element": {"ПолеПримечание"}, "_event": {"ПриИзменении"}, "Наименование": {"Тест"},
		"Примечание": {"x"}, "СкрытьПоле": {fmt.Sprint(carried)},
	}).Body.Bytes())
	if second.ElementStates == nil || !second.ElementStates.Hidden["ПолеСкрываемое"] {
		t.Fatalf("перевезённый флаг потерян: поле снова видно после постороннего события: %+v", second.ElementStates)
	}

	// Без скрытого поля (как было) то же событие откатывает условие — это и
	// есть дефект, который закрывает перевозка.
	bare := decodeFormEventResponse(t, executeFormEvent(t, s, ent, url.Values{
		"_element": {"ПолеПримечание"}, "_event": {"ПриИзменении"}, "Наименование": {"Тест"}, "Примечание": {"x"},
	}).Body.Bytes())
	if bare.ElementStates == nil || bare.ElementStates.Hidden["ПолеСкрываемое"] {
		t.Fatalf("без перевозки поле должно было открыться, контроль сломан: %+v", bare.ElementStates)
	}
}

// Значение реквизита формы на момент отрисовки уходит в конфиг managed.js:
// из него клиент заполняет скрытое поле до первого события.
func TestManagedFormConfigCarriesFormAttrValues(t *testing.T) {
	_, ent := carrierEventServer(t)
	html := renderManagedForm(t, ent, ent.Forms[0], map[string]string{
		"Наименование": "Тест", "СкрытьПоле": "true",
	})
	m := regexp.MustCompile(`(?s)<script type="application/json" id="ob-managed-config">(.*?)</script>`).FindStringSubmatch(html)
	if m == nil {
		t.Fatal("нет ob-managed-config")
	}
	var cfg struct {
		FormAttrs      []string       `json:"formAttrs"`
		FormAttrValues map[string]any `json:"formAttrValues"`
	}
	if err := json.Unmarshal([]byte(m[1]), &cfg); err != nil {
		t.Fatalf("конфиг формы: %v\n%s", err, m[1])
	}
	if len(cfg.FormAttrs) != 3 {
		t.Fatalf("formAttrs = %v, ожидались три реквизита формы", cfg.FormAttrs)
	}
	if got := cfg.FormAttrValues["СкрытьПоле"]; got != "true" {
		t.Fatalf("formAttrValues[СкрытьПоле] = %#v, ожидалось \"true\" (всё: %v)", got, cfg.FormAttrValues)
	}
	if _, ok := cfg.FormAttrValues["Примечание"]; ok {
		t.Fatalf("незаполненный реквизит попал в formAttrValues: %v", cfg.FormAttrValues)
	}
}

// Клиентская половина: скрытые поля заводятся только реквизитам без своего
// контрола и до восстановления/первого события (managed.js).
func TestFormAttrCarriersBehavior(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is required for the form attribute carrier regression test")
	}
	cmd := exec.Command(node, "--test", "static/form_attr_carriers_behavior_test.js") //nolint:gosec // test-only executable resolved by exec.LookPath
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("node form attribute carrier test: %v\n%s", err, output)
	}
}
