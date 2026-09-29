package service

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestGenerateCode_Length(t *testing.T) {
	code, err := generateCode(codeLength)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(code) != codeLength {
		t.Fatalf("want length %d, got %d", codeLength, len(code))
	}
}

func TestGenerateCode_Alphabet(t *testing.T) {
	for i := 0; i < 200; i++ {
		code, err := generateCode(codeLength)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, ch := range code {
			if !strings.ContainsRune(codeAlphabet, ch) {
				t.Fatalf("unexpected char %q in code %q", ch, code)
			}
		}
	}
}

func TestGenerateCode_Uniqueness(t *testing.T) {
	seen := make(map[string]struct{}, 1000)
	for i := 0; i < 1000; i++ {
		code, err := generateCode(codeLength)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := seen[code]; ok {
			t.Fatalf("duplicate code generated: %q", code)
		}
		seen[code] = struct{}{}
	}
}

func TestCreateLink_InvalidURL(t *testing.T) {
	svc := &LinkService{}
	cases := []string{
		"",
		"not a url",
		"ftp://example.com",
		"//example.com",
		"javascript:alert(1)",
	}
	for _, c := range cases {
		_, err := svc.CreateLink(context.Background(), c)
		if !errors.Is(err, ErrInvalidURL) {
			t.Errorf("url=%q: want ErrInvalidURL, got %v", c, err)
		}
	}
}
