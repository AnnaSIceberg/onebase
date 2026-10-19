package auth_test

import (
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ivantit66/onebase/internal/auth"
)

// Страница входа со списком пользователей: свой список вместо <datalist> (тот
// при подставленном браузером логине показывал одного пользователя), поле
// пароля с id — его очищает смена имени.
func TestLoginPage_UserPickerReplacesDatalist(t *testing.T) {
	repo, ctx := newTestRepo(t)
	for _, login := range []string{"Сотрудник ОТК Тест", "ivan"} {
		u, err := repo.Create(ctx, login, "password", "", false)
		if err != nil {
			t.Fatal(err)
		}
		if err := repo.SetShowInList(ctx, u.ID, true); err != nil {
			t.Fatal(err)
		}
	}
	h := &auth.Handlers{Repo: repo}
	rec := httptest.NewRecorder()
	h.LoginPage(rec, httptest.NewRequest("GET", "/login", nil))
	page := rec.Body.String()
	for _, want := range []string{
		`id="ob-login"`, `role="combobox"`, `id="ob-users-toggle"`, `id="ob-password"`,
		`data-login="Сотрудник ОТК Тест"`, `data-login="ivan"`, `function filterUsers`,
	} {
		if !strings.Contains(page, want) {
			t.Fatalf("на странице входа нет %q", want)
		}
	}
	if strings.Contains(page, "<datalist") || strings.Contains(page, `list="ob-users"`) {
		t.Fatal("страница входа всё ещё отдаёт <datalist>")
	}
}

// Без пользователей в списке — ни кнопки, ни скрипта: поле обычное.
func TestLoginPage_NoUsersNoPicker(t *testing.T) {
	repo, _ := newTestRepo(t)
	rec := httptest.NewRecorder()
	(&auth.Handlers{Repo: repo}).LoginPage(rec, httptest.NewRequest("GET", "/login", nil))
	page := rec.Body.String()
	if strings.Contains(page, `id="ob-users-toggle"`) || strings.Contains(page, "function filterUsers") {
		t.Fatal("без пользователей список не нужен")
	}
	if !strings.Contains(page, `name="login"`) || !strings.Contains(page, `name="password"`) {
		t.Fatal("поля входа пропали")
	}
}

// Логика списка — в node: все пользователи по стрелке при подставленном
// логине, фильтр по набранному, переход по кругу, пароль — только при смене
// имени человеком.
func TestLoginUserPickerLogicInNode(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not installed")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "picker.js"), []byte(auth.LoginUserPickerJSForTest), 0o600); err != nil {
		t.Fatal(err)
	}
	test := `
const assert = require('node:assert');
const p = require('./picker.js');
const users = [{login: 'Сотрудник ОТК Тест', name: 'Сотрудник ОТК Тест'}, {login: 'ivan', name: 'Иван Петров'}, {login: 'olga', name: ''}];
assert.strictEqual(p.filterUsers(users, 'Сотрудник ОТК Тест', true).length, 3, 'стрелка — все, даже при подставленном логине');
assert.deepStrictEqual(p.filterUsers(users, 'пет', false).map(u => u.login), ['ivan'], 'фильтр по имени без учёта регистра');
assert.deepStrictEqual(p.filterUsers(users, 'OL', false).map(u => u.login), ['olga'], 'фильтр по логину');
assert.strictEqual(p.filterUsers(users, '  ', false).length, 3, 'пустой набор — все');
assert.strictEqual(p.moveIndex(-1, 1, 3), 0);
assert.strictEqual(p.moveIndex(-1, -1, 3), 2);
assert.strictEqual(p.moveIndex(2, 1, 3), 0, 'по кругу вниз');
assert.strictEqual(p.moveIndex(0, -1, 3), 2, 'по кругу вверх');
assert.strictEqual(p.moveIndex(0, 1, 0), -1, 'пустой список');
assert.strictEqual(p.shouldClearPassword('a', 'b', true), true, 'сменил человек — пароль очищается');
assert.strictEqual(p.shouldClearPassword('a', 'b', false), false, 'подставил браузер — пароль остаётся');
assert.strictEqual(p.shouldClearPassword('a', 'a', true), false, 'то же имя — пароль остаётся');
console.log('ok');
`
	if err := os.WriteFile(filepath.Join(dir, "picker_test.js"), []byte(test), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(node, "picker_test.js") //nolint:gosec // test-only executable resolved by exec.LookPath
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("node: %v\n%s", err, out)
	}
}
