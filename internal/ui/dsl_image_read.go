package ui

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dsl/interpreter"
	"github.com/ivantit66/onebase/internal/metadata"
)

func (s *Server) readStoredImageBytes(ctx context.Context, args []any) (any, error) {
	if auth.UserFromContext(ctx) != nil {
		return nil, fmt.Errorf("ПрочитатьБайтыКартинки: доступно только серверному заданию")
	}
	if len(args) != 2 {
		return nil, fmt.Errorf("ПрочитатьБайтыКартинки: ожидаются ссылка и имя поля")
	}
	ref, refOK := args[0].(*interpreter.Ref)
	fieldName, fieldOK := args[1].(string)
	if !refOK || ref == nil || !fieldOK {
		return nil, fmt.Errorf("ПрочитатьБайтыКартинки: ожидаются ссылка и имя поля")
	}
	entity := s.reg.GetEntity(ref.Type)
	if entity == nil {
		return nil, nil
	}
	field := findObjectAttributeField(entity, fieldName)
	if field == nil || field.Type != metadata.FieldTypeImage {
		return nil, fmt.Errorf("ПрочитатьБайтыКартинки: ожидается поле типа image")
	}
	value, err := s.objectAttributeValue(ctx, args)
	if err != nil {
		return nil, err
	}
	raw, ok := value.(string)
	if !ok || raw == "" {
		return nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, nil
	}
	blob, reader, err := s.store.OpenBlob(ctx, id)
	if err != nil {
		return nil, err
	}
	defer closeRead("Изображение из хранилища", reader)

	const limit = 8 << 20
	if !strings.HasPrefix(blob.Mime, "image/") || blob.OwnerKind != string(entity.Kind) || blob.OwnerEntity != entity.Name {
		return nil, nil
	}
	if blob.Size > limit {
		return nil, fmt.Errorf("ПрочитатьБайтыКартинки: изображение превышает 8 МиБ")
	}
	body, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if len(body) > limit || int64(len(body)) != blob.Size {
		return nil, fmt.Errorf("ПрочитатьБайтыКартинки: неверный размер изображения")
	}
	return string(body), nil
}
