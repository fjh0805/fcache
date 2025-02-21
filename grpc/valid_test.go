package grpc

import "testing"

func TestValid(t *testing.T) {
	defaultaddr := "127.0.0.1:6324"

	if !validPeerAddr(defaultaddr) {
		t.Fatalf("defaultaddr is right addr")
	}
}
