package ui

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/ivantit66/onebase/internal/dsl/interpreter"
)

// Preserve the historical user message while carrying a typed native denial.
// A user-raised error with the same text must not become an authorization result.
type querySourceAccessDenied struct{ source string }

func (e *querySourceAccessDenied) Error() string {
	return fmt.Sprintf("нет доступа к объекту: %s", e.source)
}
func (e *querySourceAccessDenied) Unwrap() error { return interpreter.ErrRowAccessDenied }

type reportAccessRefused struct{ message string }

func (e *reportAccessRefused) Error() string { return e.message }

func refuseReportAccess(args []any, _ string, _ int) (any, error) {
	if len(args) != 1 {
		return nil, errors.New("ОтказатьВДоступеКОтчету: ожидается текст отказа")
	}
	message, ok := args[0].(string)
	if !ok || strings.TrimSpace(message) == "" {
		return nil, errors.New("ОтказатьВДоступеКОтчету: ожидается непустой текст отказа")
	}
	return nil, &reportAccessRefused{message: strings.TrimSpace(message)}
}

func (s *Server) writeTabularReportError(w http.ResponseWriter, r *http.Request, err error) {
	var refused *reportAccessRefused
	if errors.As(err, &refused) {
		http.Error(w, refused.message, http.StatusForbidden)
		return
	}
	http.Error(w, s.errText(r, err), reportResultErrorStatus(err))
}

func reportResultErrorStatus(err error) int {
	if errors.Is(err, errReportFilterProducerUnproven) {
		return http.StatusUnprocessableEntity
	}
	if errors.Is(err, interpreter.ErrRowAccessDenied) {
		return http.StatusForbidden
	}
	return http.StatusInternalServerError
}
