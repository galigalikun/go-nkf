package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestGuess(t *testing.T) {
	// Encoding checks must work even when no external commands are available.
	t.Setenv("PATH", t.TempDir())

	tests := []struct {
		file    string
		want    string
		wantErr bool
	}{
		{file: "test.euc-jp.txt", wantErr: true},
		{file: "test.shift-jis.txt", wantErr: true},
		{file: "test.utf-8.txt", want: "UTF-8 (LF)\n"},
		{file: "test.txt", want: "ASCII (LF)\n"},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			path := filepath.Join("testdata", tt.file)
			got, err := guess(path)
			if got != tt.want || (err != nil) != tt.wantErr {
				t.Errorf("guess() = %q, %v; want %q, error=%t", got, err, tt.want, tt.wantErr)
			}
			if err := guessenc(path); (err != nil) != tt.wantErr {
				t.Errorf("guessenc() = %v; want error=%t", err, tt.wantErr)
			}
		})
	}
}

func TestGuessContent(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    string
		wantErr bool
	}{
		{name: "empty", want: "ASCII\n"},
		{name: "ascii", data: "hello", want: "ASCII\n"},
		{name: "utf8", data: "日本語😀", want: "UTF-8\n"},
		{name: "replacement character", data: "\ufffd", want: "UTF-8\n"},
		{name: "bom only", data: "\xef\xbb\xbf", want: "UTF-8 (BOM)\n"},
		{name: "bom ascii", data: "\xef\xbb\xbfhello", want: "UTF-8 (BOM)\n"},
		{name: "bom utf8", data: "\xef\xbb\xbf日本語\n", want: "UTF-8 (BOM) (LF)\n"},
		{name: "lf", data: "a\nb\n", want: "ASCII (LF)\n"},
		{name: "crlf", data: "a\r\nb\r\n", want: "ASCII (CRLF)\n"},
		{name: "cr", data: "a\rb\r", want: "ASCII (CR)\n"},
		{name: "trailing cr", data: "a\r", want: "ASCII (CR)\n"},
		{name: "mixed lf crlf", data: "a\nb\r\n", want: "ASCII (MIXED NL)\n"},
		{name: "mixed cr crlf", data: "a\rb\r\n", want: "ASCII (MIXED NL)\n"},
		{name: "mixed lf cr", data: "a\n\r", want: "ASCII (MIXED NL)\n"},
		{name: "invalid byte", data: "hello\xff", wantErr: true},
		{name: "truncated utf8", data: "\xe3\x81", wantErr: true},
		{name: "stray continuation", data: "\x80", wantErr: true},
		{name: "overlong utf8", data: "\xc0\xaf", wantErr: true},
		{name: "surrogate", data: "\xed\xa0\x80", wantErr: true},
		{name: "out of range", data: "\xf4\x90\x80\x80", wantErr: true},
		{name: "utf16 bom", data: "\xff\xfeA\x00", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "input.txt")
			if err := os.WriteFile(path, []byte(tt.data), 0600); err != nil {
				t.Fatal(err)
			}
			got, err := guess(path)
			if got != tt.want || (err != nil) != tt.wantErr {
				t.Errorf("guess() = %q, %v; want %q, error=%t", got, err, tt.want, tt.wantErr)
			}
			if err := guessenc(path); (err != nil) != tt.wantErr {
				t.Errorf("guessenc() = %v; want error=%t", err, tt.wantErr)
			}
		})
	}
}

func TestGuessReadErrors(t *testing.T) {
	dir := t.TempDir()
	for _, path := range []string{filepath.Join(dir, "missing.txt"), dir} {
		t.Run(filepath.Base(path), func(t *testing.T) {
			got, err := guess(path)
			if got != "" || err == nil {
				t.Fatalf("guess() = %q, %v; want empty result and read error", got, err)
			}
			var pathErr *os.PathError
			if !errors.As(err, &pathErr) {
				t.Errorf("guess() error = %T; want *os.PathError", err)
			}
			if err := guessenc(path); !errors.As(err, &pathErr) {
				t.Errorf("guessenc() error = %v; want *os.PathError", err)
			}
		})
	}
}
