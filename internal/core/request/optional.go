package request

import "encoding/json"

// Optional — поле JSON с тремя состояниями: его нет в запросе, оно равно null или у него есть значение
// Нужно для PATCH: «поле не прислали» значит «не менять», а null значит «очистить»
type Optional[T any] struct {
	Set   bool
	Value *T
}

// UnmarshalJSON вызывается пакетом json, только если поле есть в запросе
func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	o.Set = true

	if string(data) == "null" {
		o.Value = nil
		return nil
	}

	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	o.Value = &v

	return nil
}
