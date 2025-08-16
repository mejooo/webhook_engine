package fastpath

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"

	"webhook-engine/pkg/events"
	"webhook-engine/pkg/fastqueue"
	"webhook-engine/pkg/metrics"
)

type Validator struct {
	Token []byte
	In    <-chan fastqueue.Event
	Out   chan<- []byte
}

func (v *Validator) Run() {
	for e := range v.In {
		if verifyV0(e.TS, e.Body, v.Token, e.Sig) {
			metrics.ValidatedTotal.Inc()
			val := events.Valid{ Raw: events.Raw{ Source: "zoom", Format: "json", Body: e.Body } }
			if b := events.MarshalValid(val); b != nil {
				v.Out <- b
			}
		} else {
			metrics.InvalidTotal.Inc()
		}
	}
}

func verifyV0(ts, body, token, sig []byte) bool {
	msg := make([]byte, 0, 3+len(ts)+1+len(body))
	msg = append(msg, 'v','0',':')
	msg = append(msg, ts...)
	msg = append(msg, ':')
	msg = append(msg, body...)
	h := hmac.New(sha256.New, token); h.Write(msg)
	sum := h.Sum(nil)
	hexBuf := make([]byte, hex.EncodedLen(len(sum))); hex.Encode(hexBuf, sum)
	want := append([]byte("v0="), hexBuf...)
	return subtle.ConstantTimeCompare(want, sig) == 1
}
