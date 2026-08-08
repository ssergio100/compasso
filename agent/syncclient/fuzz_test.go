package syncclient

import (
	"bytes"
	"testing"
)

func FuzzHeartbeatResponseDoesNotPanic(fuzzer *testing.F) {
	fuzzer.Add([]byte(`{}`))
	fuzzer.Add([]byte(`{"policy":{"revision":-1}}`))
	fuzzer.Add([]byte(`{"commands":[{"id":"x","kind":"unknown","payload":{}}]}`))
	fuzzer.Add([]byte(`{"server_time":"not-a-time"}`))
	fuzzer.Add([]byte(`{} {}`))
	fuzzer.Fuzz(func(t *testing.T, payload []byte) {
		_, _ = decodeHeartbeatResponse(bytes.NewReader(payload))
	})
}
