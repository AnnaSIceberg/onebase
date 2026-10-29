package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	// ErrLoginTaken — новый логин уже занят другой учёткой (без учёта
	// регистра: вход по SSO ищет учётку именно так).
	ErrLoginTaken = errors.New("логин уже занят другим пользователем")
	// ErrSameLogin — переименование в тот же логин.
	ErrSameLogin = errors.New("новый логин совпадает с прежним")
)

const maxLoginRunes = 64

// ValidateLogin проверяет новый логин: буквы любого алфавита (кириллица
// тоже), цифры и «_ . - @», без пробелов, до 64 символов. «Оператор_075» и
// «y.s.lagutin» допустимы.
func ValidateLogin(login string) error {
	if login == "" {
		return errors.New("логин не задан")
	}
	if !utf8.ValidString(login) {
		return errors.New("логин: недопустимая кодировка")
	}
	if utf8.RuneCountInString(login) > maxLoginRunes {
		return fmt.Errorf("логин длиннее %d символов", maxLoginRunes)
	}
	for _, r := range login {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune("_.-@", r) {
			continue
		}
		return fmt.Errorf("логин: недопустимый символ %q — можно буквы, цифры и «_ . - @»", r)
	}
	return nil
}

// RenameUser переименовывает учётку oldLogin в newLogin одной транзакцией:
//   - _users.login;
//   - личные настройки и варианты отчётов, хранящиеся по логину строкой
//     (storage.RenameUserData);
//   - все сессии учётки завершаются: в них записан прежний логин, и войти
//     нужно заново.
//
// Пароль, второй фактор, резервные коды, роли и API-токены привязаны к id и
// сохраняются. Журналы не меняются: они история. Данные конфигурации, где
// логин записан строкой (например, регистры сведений), платформа не знает —
// их переносит сама конфигурация.
func (r *Repo) RenameUser(ctx context.Context, oldLogin, newLogin string) (*User, error) {
	if err := ValidateLogin(newLogin); err != nil {
		return nil, err
	}
	if oldLogin == newLogin {
		return nil, ErrSameLogin
	}
	d := r.db.Dialect()
	var u *User
	err := r.withUserInvariantLock(ctx, func(txCtx context.Context) error {
		var id string
		q := fmt.Sprintf(`SELECT id FROM _users WHERE login = %s`, d.Placeholder(1))
		if err := r.db.QueryRow(txCtx, q, oldLogin).Scan(&id); err != nil {
			return fmt.Errorf("пользователь %q не найден", oldLogin)
		}
		var taken int
		q = fmt.Sprintf(`SELECT count(*) FROM _users WHERE lower(login) = lower(%s) AND id <> %s`, d.Placeholder(1), d.Placeholder(2))
		if err := r.db.QueryRow(txCtx, q, newLogin, id).Scan(&taken); err != nil {
			return err
		}
		if taken > 0 {
			return ErrLoginTaken
		}
		q = fmt.Sprintf(`UPDATE _users SET login = %s WHERE id = %s`, d.Placeholder(1), d.Placeholder(2))
		if _, err := r.db.Exec(txCtx, q, newLogin, id); err != nil {
			return fmt.Errorf("auth: rename user: %w", err)
		}
		if err := r.db.RenameUserData(txCtx, oldLogin, newLogin); err != nil {
			return err
		}
		if _, err := r.db.Exec(txCtx, `DELETE FROM _sessions WHERE user_id = `+d.Placeholder(1), id); err != nil {
			return fmt.Errorf("auth: rename user: sessions: %w", err)
		}
		got, err := r.GetByID(txCtx, id)
		if err != nil {
			return err
		}
		u = got
		return nil
	})
	if err != nil {
		return nil, err
	}
	return u, nil
}
