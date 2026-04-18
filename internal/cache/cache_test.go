package cache

import (
	"testing"
	"time"
)

func TestCacheSetGetAndExpiry(t *testing.T) {
	c := NewMemory(50 * time.Millisecond)
	c.Set("k", []byte("v"))

	got, ok := c.Get("k")
	if !ok {
		t.Fatalf("expected key to exist")
	}
	if string(got) != "v" {
		t.Fatalf("unexpected value %q", string(got))
	}

	time.Sleep(70 * time.Millisecond)
	if _, ok := c.Get("k"); ok {
		t.Fatalf("expected key to expire")
	}
}

