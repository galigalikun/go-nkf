package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIConversions(t *testing.T) {
	const sjis = "\x93\xfa\x96\x7b\x8c\xea"
	const euc = "\xc6\xfc\xcb\xdc\xb8\xec"
	const jis = "\x1b$BF|K\\8l\x1b(B"
	tests := []struct {
		name  string
		args  []string
		input string
		want  string
	}{
		{"default jis output", nil, "日本語", jis},
		{"utf8 stdin", []string{"-w"}, "日本語\n", "日本語\n"},
		{"sjis auto", []string{"-w"}, sjis, "日本語"},
		{"euc auto", []string{"-w"}, euc, "日本語"},
		{"jis auto", []string{"-w"}, jis, "日本語"},
		{"sjis output", []string{"-Ws"}, "日本語", sjis},
		{"euc output", []string{"-We"}, "日本語", euc},
		{"jis output", []string{"-Wj"}, "日本語", jis},
		{"explicit sjis", []string{"-Sw"}, sjis, "日本語"},
		{"explicit euc", []string{"-Ew"}, euc, "日本語"},
		{"explicit jis", []string{"-Jw"}, jis, "日本語"},
		{"long encodings", []string{"--ic=cp932", "--oc=UTF-8"}, sjis, "日本語"},
		{"separate encoding values", []string{"--ic", "utf_8", "--oc", "Windows-31J"}, "日本語", sjis},
		{"long aliases", []string{"--sjis-input", "--euc"}, sjis, euc},
		{"long jis alias", []string{"--euc-input", "--jis"}, euc, jis},
		{"long sjis alias", []string{"--jis-input", "--sjis"}, jis, sjis},
		{"combined newline", []string{"-SwLu"}, sjis + "\r\n", "日本語\n"},
		{"mixed to lf", []string{"-wLu"}, "a\r\nb\rc\n", "a\nb\nc\n"},
		{"mixed to crlf", []string{"-wLw"}, "a\r\nb\rc\n", "a\r\nb\r\nc\r\n"},
		{"mixed to cr", []string{"-wLm"}, "a\r\nb\rc\n", "a\rb\rc\r"},
		{"newline preserve", []string{"-wLuL0"}, "a\r\nb\r", "a\r\nb\r"},
		{"delete cr alias", []string{"-wd"}, "a\r\n", "a\n"},
		{"add cr alias", []string{"-wc"}, "a\n", "a\r\n"},
		{"remove bom", []string{"-w"}, "\xef\xbb\xbf日本語", "日本語"},
		{"utf8 bom", []string{"-w8"}, "日本語", "\xef\xbb\xbf日本語"},
		{"utf8 no bom", []string{"-w80"}, "\xef\xbb\xbf日本語", "日本語"},
		{"long utf8 bom", []string{"--oc=UTF-8-BOM"}, "A", "\xef\xbb\xbfA"},
		{"forced utf8 bom", []string{"-Ww8"}, "\xef\xbb\xbfA", "\xef\xbb\xbfA"},
		{"utf16 be", []string{"-w16"}, "Aあ😀\n", "\xfe\xff\x00A\x30\x42\xd8\x3d\xde\x00\x00\x0a"},
		{"utf16 le", []string{"-w16L"}, "Aあ😀\n", "\xff\xfeA\x00\x42\x30\x3d\xd8\x00\xde\x0a\x00"},
		{"utf16 without bom", []string{"-w16B0"}, "A", "\x00A"},
		{"utf16 auto input", []string{"-w"}, "\xff\xfeA\x00\x42\x30\x3d\xd8\x00\xde", "Aあ😀"},
		{"utf16 explicit input", []string{"-W16Lw"}, "A\x00\x42\x30", "Aあ"},
		{"utf16 bom endian", []string{"-W16Bw"}, "\xff\xfeA\x00", "A"},
		{"utf16 long default", []string{"--oc=UTF-16"}, "A", "\x00A"},
		{"utf32 be", []string{"-w32"}, "A😀", "\x00\x00\xfe\xff\x00\x00\x00A\x00\x01\xf6\x00"},
		{"utf32 le", []string{"-w32L"}, "A😀", "\xff\xfe\x00\x00A\x00\x00\x00\x00\xf6\x01\x00"},
		{"utf32 no bom", []string{"-w32L0"}, "A", "A\x00\x00\x00"},
		{"utf32 auto input", []string{"-w"}, "\xff\xfe\x00\x00A\x00\x00\x00", "A"},
		{"utf32 explicit input", []string{"-W32Bw"}, "\x00\x01\xf6\x00", "😀"},
		{"widen kana", []string{"-w"}, "abc ﾊﾝｶｸ ｶﾞﾊﾟ｡", "abc ハンカク ガパ。"},
		{"preserve kana", []string{"-wx"}, "abc ｶﾞ", "abc ｶﾞ"},
		{"widen kana override", []string{"-wxX"}, "ｶﾞ", "ガ"},
		{"sjis halfwidth explicit", []string{"-Swx"}, "\xb6\xde", "ｶﾞ"},
		{"ascii output", []string{"--oc=ASCII"}, "hello\n", "hello\n"},
		{"copy binary", []string{"-t"}, "\x00\xff\r\n", "\x00\xff\r\n"},
		{"literal replacement", []string{"-w"}, "\ufffd", "\ufffd"},
		{"empty", []string{"-w"}, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			if err := runCLI(tt.args, strings.NewReader(tt.input), &output); err != nil {
				t.Fatal(err)
			}
			if got := output.String(); got != tt.want {
				t.Errorf("output = %q (%x), want %q (%x)", got, got, tt.want, tt.want)
			}
		})
	}
}

func TestCLIGuess(t *testing.T) {
	tests := []struct {
		args  []string
		input string
		want  string
	}{
		{[]string{"-g"}, "日本語\n", "UTF-8\n"},
		{[]string{"--guess"}, "日本語\r\n", "UTF-8 (CRLF)\n"},
		{[]string{"--guess=1"}, "\xef\xbb\xbfA\n", "UTF-8\n"},
		{[]string{"--guess=2"}, "\xef\xbb\xbfA\n", "UTF-8 (BOM) (LF)\n"},
		{[]string{"-g2"}, "\xff\xfeA\x00\x0a\x00", "UTF-16 LE (BOM) (LF)\n"},
		{[]string{"-g"}, "\xff\xfe\x00\x00A\x00\x00\x00", "UTF-32\n"},
		{[]string{"-g2"}, "a\r\nb\n", "ASCII (MIXED NL)\n"},
		{[]string{"--guess"}, "\xff", "BINARY\n"},
		{[]string{"--guess"}, "\xfe\xff\xd8\x00", "BINARY\n"},
		{[]string{"--ic=UTF-8-BOM", "--guess"}, "abc", "UTF-8\n"},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " ")+tt.want, func(t *testing.T) {
			var output bytes.Buffer
			if err := runCLI(tt.args, strings.NewReader(tt.input), &output); err != nil {
				t.Fatal(err)
			}
			if got := output.String(); got != tt.want {
				t.Errorf("output = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCLIFileInputs(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	fixtures := []struct{ file, want string }{
		{"test.euc-jp.txt", "EUC-JP (LF)\n"},
		{"test.shift-jis.txt", "Shift_JIS (LF)\n"},
		{"test.utf-8.txt", "UTF-8 (LF)\n"},
		{"test.txt", "ASCII (LF)\n"},
	}
	var args []string
	args = append(args, "--guess")
	var want strings.Builder
	for _, fixture := range fixtures {
		path := filepath.Join("testdata", fixture.file)
		args = append(args, path)
		want.WriteString(path + ": " + fixture.want)
	}
	var output bytes.Buffer
	if err := runCLI(args, errorReader{}, &output); err != nil {
		t.Fatal(err)
	}
	if output.String() != want.String() {
		t.Errorf("output = %q, want %q", output.String(), want.String())
	}

	output.Reset()
	if err := runCLI([]string{"-w", "testdata/test.euc-jp.txt"}, errorReader{}, &output); err != nil {
		t.Fatal(err)
	}
	if want := "EUC-JP\n\nあああ\n\nええ\n\nほほ\n"; output.String() != want {
		t.Errorf("output = %q, want %q", output.String(), want)
	}

	output.Reset()
	if err := runCLI([]string{"-w", "testdata/test.txt", "-"}, strings.NewReader("日本語"), &output); err != nil {
		t.Fatal(err)
	}
	if want := "ababa\n\ndgasgagas\n\nbadbad\n日本語"; output.String() != want {
		t.Errorf("output = %q, want %q", output.String(), want)
	}
}

func TestCLIEndOptions(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile("-w", []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := runCLI([]string{"-w", "--", "-w"}, errorReader{}, &output); err != nil {
		t.Fatal(err)
	}
	if output.String() != "hello" {
		t.Errorf("output = %q", output.String())
	}
}

func TestCLIMultipleFilesOneBOM(t *testing.T) {
	dir := t.TempDir()
	first, second := filepath.Join(dir, "first"), filepath.Join(dir, "second")
	if err := os.WriteFile(first, []byte("あ"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte("い"), 0600); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := runCLI([]string{"-w8", first, second}, errorReader{}, &output); err != nil {
		t.Fatal(err)
	}
	if want := "\xef\xbb\xbfあい"; output.String() != want {
		t.Errorf("output = %q, want %q", output.String(), want)
	}
}

func TestCLIInvalidOptions(t *testing.T) {
	for _, args := range [][]string{
		{"--unknown"}, {"-q"}, {"--overwrite"}, {"-w64"}, {"-W80"},
		{"--ic"}, {"--oc="}, {"--ic", "--guess"}, {"--ic=invalid"},
		{"--guess=3"}, {"--guess="}, {"-g3"}, {"-L"}, {"-Lz"},
		{"--jis=value"}, {"-tg"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var output bytes.Buffer
			if err := runCLI(args, errorReader{}, &output); err == nil {
				t.Fatal("expected argument error")
			}
			if output.Len() != 0 {
				t.Errorf("unexpected output: %q", output.String())
			}
		})
	}
}

func TestCLIConversionErrors(t *testing.T) {
	tests := []struct {
		args  []string
		input string
	}{
		{[]string{"-w"}, "\xff"},
		{[]string{"-Ww"}, "\xe3\x81"},
		{[]string{"-Sw"}, "\x82"},
		{[]string{"-Ew"}, "\xa4"},
		{[]string{"-Jw"}, "\x1b$B!"},
		{[]string{"-W16Lw"}, "A"},
		{[]string{"-W16Lw"}, "\x00\xd8"},
		{[]string{"-W16Lw"}, "\x00\xdc"},
		{[]string{"-W16Lw"}, "\x00\xd8A\x00"},
		{[]string{"-W32Bw"}, "\x00\x11\x00\x00"},
		{[]string{"-W32Bw"}, "\x00\x00\xd8\x00"},
		{[]string{"-W32Bw"}, "\x00\x00"},
		{[]string{"-Ww"}, "\xff\xfeA\x00"},
		{[]string{"-w"}, "A\x00"},
		{[]string{"--ic=ASCII", "-w"}, "日本語"},
		{[]string{"--oc=ASCII"}, "日本語"},
		{[]string{"-s"}, "😀"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s/%x", strings.Join(tt.args, " "), tt.input), func(t *testing.T) {
			var output bytes.Buffer
			if err := runCLI(tt.args, strings.NewReader(tt.input), &output); err == nil {
				t.Fatal("expected conversion error")
			}
			if output.Len() != 0 {
				t.Errorf("unexpected output: %q", output.String())
			}
		})
	}
}

var errTestIO = errors.New("test I/O failure")

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) { return 0, errTestIO }

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) { return 0, errTestIO }

type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) { return len(p) / 2, nil }

func TestCLIIOErrors(t *testing.T) {
	if err := runCLI([]string{"-w"}, errorReader{}, io.Discard); !errors.Is(err, errTestIO) {
		t.Errorf("stdin error = %v", err)
	}
	if err := runCLI([]string{"-w"}, strings.NewReader("abc"), errorWriter{}); !errors.Is(err, errTestIO) {
		t.Errorf("stdout error = %v", err)
	}
	if err := runCLI([]string{"-w"}, strings.NewReader("abc"), shortWriter{}); !errors.Is(err, io.ErrShortWrite) {
		t.Errorf("short write error = %v", err)
	}
	var output bytes.Buffer
	missing := filepath.Join(t.TempDir(), "missing")
	if err := runCLI([]string{"-w", missing, "-"}, strings.NewReader("abc"), &output); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("missing file error = %v", err)
	}
	if output.String() != "abc" {
		t.Errorf("remaining input was not processed: %q", output.String())
	}
}

func TestCLIHelpVersion(t *testing.T) {
	for _, arg := range []string{"--help", "--version", "-v"} {
		var output bytes.Buffer
		if err := runCLI([]string{arg}, errorReader{}, &output); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(output.String(), "go-nkf") {
			t.Errorf("%s output = %q", arg, output.String())
		}
	}
}
