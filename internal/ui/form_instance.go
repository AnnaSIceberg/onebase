package ui

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dsl/interpreter"
	"github.com/ivantit66/onebase/internal/metadata"
)

const managedFormInstanceHeader = "X-OneBase-Form-Instance"
const managedFormInstanceTTL = 2 * time.Hour

func (s *Server) issueManagedFormInstance(r *http.Request, data map[string]any) {
	if _, issued := data["FormInstance"]; issued {
		return
	}
	form, _ := data["Form"].(*metadata.FormModule)
	if form == nil || (!form.InstanceContext && form.SavedPhoneMatch == nil) {
		return
	}
	schema, _ := data["FormCloseSchema"].(string)
	var scope string
	if processor, _ := data["IsProcessor"].(bool); processor {
		scope = r.URL.Query().Get(resultContextQuery)
	} else if isNew, _ := data["IsNew"].(bool); !isNew {
		scope = fmt.Sprint(data["ID"])
	}
	if token := s.newManagedFormInstance(r, schema, scope, time.Now()); token != "" {
		data["FormInstance"] = token
		entity, _ := data["Entity"].(*metadata.Entity)
		nonce, _, _ := strings.Cut(token, ".")
		if s.bindFormOpenContext(r, entity, scope, nonce, s.concealNow()) {
			data["FormOpeningInstanceField"] = entityFormServiceFieldNames(entity, form)[formOpenInstanceField]
		}
	}
}

func (s *Server) newManagedFormInstance(r *http.Request, schema, scope string, now time.Time) string {
	nonce := uuid.NewString()
	if r.Method == http.MethodGet && r.URL.Query().Has(formOpenSavedPhoneQuery) {
		nonce = savedPhoneOpeningInstancePrefix + nonce
	}
	payload := nonce + "." + strconv.FormatInt(now.Add(managedFormInstanceTTL).Unix(), 10)
	signature := s.managedFormInstanceSignature(r, schema, scope, payload)
	if signature == "" {
		return ""
	}
	return payload + "." + signature
}

func (s *Server) managedFormInstanceSignature(r *http.Request, schema, scope, payload string) string {
	if schema == "" || auth.UserFromContext(r.Context()) == nil {
		return ""
	}
	owner, hasSession := concealOwner(r)
	authorization, valid := formCloseAuthorizationFingerprint(r.Context())
	if !hasSession || !valid {
		return ""
	}
	mac := hmac.New(sha256.New, s.formIdleKey())
	mac.Write([]byte("onebase.managed-form-instance.v1\x00"))
	mac.Write(owner[:])
	mac.Write(authorization[:])
	for _, part := range []string{schema, scope, strconv.FormatUint(s.reg.Generation(), 10), payload} {
		writeFormCloseFingerprintString(mac, part)
	}
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *Server) validManagedFormInstance(r *http.Request, scope, schema string) (nonce, expiry string) {
	tokens := r.Header.Values(managedFormInstanceHeader)
	if len(tokens) != 1 || len(tokens[0]) > 128 {
		return "", ""
	}
	nonce, rest, ok := strings.Cut(tokens[0], ".")
	expiry, proof, hasProof := strings.Cut(rest, ".")
	expires, err := strconv.ParseInt(expiry, 10, 64)
	if !ok || !hasProof || err != nil || !time.Now().Before(time.Unix(expires, 0)) {
		return "", ""
	}
	want := s.managedFormInstanceSignature(r, schema, scope, nonce+"."+expiry)
	if want == "" || !hmac.Equal([]byte(want), []byte(proof)) {
		return "", ""
	}
	return nonce, expiry
}

func (s *Server) managedFormInstanceBuiltin(r *http.Request, form *metadata.FormModule, scope string, schema func() string) interpreter.BuiltinFunc {
	return func(args []any, _ string, _ int) (any, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("ИдЭкземпляраФормы: аргументы не ожидаются")
		}
		if form == nil || !form.InstanceContext || r.Header.Get(managedFormInstanceHeader) == "" {
			return "", nil
		}
		// Only the signed nonce is an identity. The bearer never enters DSL.
		nonce, _ := s.validManagedFormInstance(r, scope, schema())
		return nonce, nil
	}
}

// Only a confirmed insert can move a new form's existing identity to that row.
func (s *Server) rebindManagedFormInstance(r *http.Request, form *metadata.FormModule, savedID string, schema func() string) string {
	if form == nil || !form.InstanceContext || savedID == "" || r.Header.Get(managedFormInstanceHeader) == "" {
		return ""
	}
	if id, err := uuid.Parse(savedID); err != nil || id == uuid.Nil {
		return ""
	}
	formSchema := schema()
	nonce, expiry := s.validManagedFormInstance(r, "", formSchema)
	if nonce == "" {
		return ""
	}
	payload := nonce + "." + expiry
	signature := s.managedFormInstanceSignature(r, formSchema, savedID, payload)
	if signature == "" {
		return ""
	}
	return payload + "." + signature
}
