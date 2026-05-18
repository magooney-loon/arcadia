package cache_test

import (
	"testing"
	"time"

	"arcadia/internal/server/cache"
)

func TestSetAndGet(t *testing.T) {
	s := cache.New()
	s.Set("key", "value", time.Minute)
	got, ok := s.Get("key")
	if !ok {
		t.Fatal("expected cache hit, got miss")
	}
	if got != "value" {
		t.Errorf("got %v, want %q", got, "value")
	}
}

func TestMiss(t *testing.T) {
	s := cache.New()
	_, ok := s.Get("missing")
	if ok {
		t.Fatal("expected cache miss, got hit")
	}
}

func TestExpiry(t *testing.T) {
	s := cache.New()
	s.Set("key", "data", time.Millisecond)
	time.Sleep(5 * time.Millisecond)
	_, ok := s.Get("key")
	if ok {
		t.Fatal("expected entry to have expired")
	}
}

func TestOverwrite(t *testing.T) {
	s := cache.New()
	s.Set("key", "first", time.Minute)
	s.Set("key", "second", time.Minute)
	got, ok := s.Get("key")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if got != "second" {
		t.Errorf("got %v, want %q", got, "second")
	}
}

func TestMultipleKeys(t *testing.T) {
	s := cache.New()
	s.Set("a", 1, time.Minute)
	s.Set("b", 2, time.Minute)

	v, ok := s.Get("a")
	if !ok || v != 1 {
		t.Errorf("key a: got (%v, %v), want (1, true)", v, ok)
	}
	v, ok = s.Get("b")
	if !ok || v != 2 {
		t.Errorf("key b: got (%v, %v), want (2, true)", v, ok)
	}
}
