package lru

import (
	"testing"
)

type String string

func (d String) Len() int {
	return len(d)
}

func add(lru *Cache) {
	lru.Add("key1", String("1"))
	lru.Add("key2", String("111"))
}

func TestAdd(t *testing.T) {
	lru := New(int64(0), nil)
	add(lru)
	v1, _ := lru.Get("key1")
	v2, _ := lru.Get("key2")
	println(string(v1.(String)), string(v2.(String)))
	println(len(string(v1.(String))), len(string(v2.(String))))
	if lru.nBytes != int64(len("key1"))+int64(len("key2"))+int64(len(string(v2.(String)))+len(string(v1.(String)))) {
		t.Fatal("expected 12 but got", lru.nBytes)
	}
}

func TestGet(t *testing.T) {
	lru := New(int64(0), nil)
	add(lru)

	if v, ok := lru.Get("key1"); !ok || string(v.(String)) != "1" {
		t.Fatalf("cache hit key1=1 failed")
	}

	if v, ok := lru.Get("key2"); !ok || string(v.(String)) != "111" {
		t.Fatalf("cache hit key2=111 failed")
	}
}
