package dto

import "encoding/json"

// Optional is a JSON field that tells "absent" apart from "null", which a
// plain pointer can't: both decode to nil. Set is true whenever the key was
// present; Value is nil when it was present as null.
type Optional[T any] struct {
	Set   bool
	Value *T
}

// UnmarshalJSON implements [json.Unmarshaler]. encoding/json only calls it
// for keys that are present (including an explicit null), so Set stays
// false for an omitted key.
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
