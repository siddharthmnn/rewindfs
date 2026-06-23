package storage

import "testing"

func TestGenerateHashSameInput(t *testing.T) {

	hash1 := GenerateHash([]byte("hello"))
	hash2 := GenerateHash([]byte("hello"))

	if hash1 != hash2 {
		t.Errorf("expected hashes to match")
	}
}

func TestGenerateHashDifferentInput(t *testing.T) {

	hash1 := GenerateHash([]byte("hello"))
	hash2 := GenerateHash([]byte("world"))

	if hash1 == hash2 {
		t.Errorf("expected hashes to be different")
	}
}

func TestGenerateHashEmptyInput(t *testing.T) {

	hash := GenerateHash([]byte(""))

	if hash == "" {
		t.Errorf("expected non-empty hash")
	}
}
