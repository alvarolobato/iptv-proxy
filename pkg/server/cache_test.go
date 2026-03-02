package server

import (
	"testing"
	"time"
)

func TestNewResponseCache_Disabled(t *testing.T) {
	c := newResponseCache(0, 10)
	if c == nil {
		t.Fatal("newResponseCache(0) returned nil")
	}
	_, ok := c.Get("x")
	if ok {
		t.Error("Get on disabled cache should miss")
	}
}

func TestResponseCache_GetSet(t *testing.T) {
	c := newResponseCache(10*time.Second, 10)
	if c == nil {
		t.Fatal("newResponseCache returned nil")
	}
	_, ok := c.Get("k")
	if ok {
		t.Error("Get before Set should miss")
	}
	c.Set("k", []byte("payload"), "text/plain")
	ent, ok := c.Get("k")
	if !ok {
		t.Fatal("Get after Set should hit")
	}
	if string(ent.payload) != "payload" || ent.contentType != "text/plain" {
		t.Errorf("entry = %q, %q", ent.payload, ent.contentType)
	}
}
