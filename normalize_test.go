package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeLineEndings(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []byte
	}{
		{
			name:     "CRLF to LF",
			input:    []byte("line1\r\nline2\r\nline3\r\n"),
			expected: []byte("line1\nline2\nline3\n"),
		},
		{
			name:     "already LF",
			input:    []byte("line1\nline2\nline3\n"),
			expected: []byte("line1\nline2\nline3\n"),
		},
		{
			name:     "mixed endings",
			input:    []byte("line1\r\nline2\nline3\r\n"),
			expected: []byte("line1\nline2\nline3\n"),
		},
		{
			name:     "empty input",
			input:    []byte(""),
			expected: []byte(""),
		},
		{
			name:     "no newlines",
			input:    []byte("single line"),
			expected: []byte("single line"),
		},
		{
			name:     "only CRLF",
			input:    []byte("\r\n"),
			expected: []byte("\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeLineEndings(tt.input)
			if string(result) != string(tt.expected) {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestNormalizeFile(t *testing.T) {
	dir := t.TempDir()

	t.Run("normalizes CRLF file", func(t *testing.T) {
		path := filepath.Join(dir, "crlf.txt")
		os.WriteFile(path, []byte("hello\r\nworld\r\n"), 0644)

		result, err := NormalizeFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(result) != "hello\nworld\n" {
			t.Errorf("got %q, want %q", result, "hello\nworld\n")
		}
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		_, err := NormalizeFile(filepath.Join(dir, "missing.txt"))
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})
}
