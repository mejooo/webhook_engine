package events

import "encoding/json"

func MarshalValid(v Valid) []byte {
	b, err := json.Marshal(v)
	if err != nil { return nil }
	return b
}
