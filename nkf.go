package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"unicode/utf8"
)

// guess validates UTF-8/ASCII and reports BOM and newline information.
// It does not attempt to identify other encodings.
func guess(fileName string) (string, error) {
	data, err := os.ReadFile(fileName)
	if err != nil {
		return "", err
	}
	if !utf8.Valid(data) {
		return "", errors.New("unsupported encoding: expected UTF-8 or ASCII")
	}

	encoding := "ASCII"
	for _, b := range data {
		if b >= utf8.RuneSelf {
			encoding = "UTF-8"
			break
		}
	}
	if bytes.HasPrefix(data, []byte{0xef, 0xbb, 0xbf}) {
		encoding += " (BOM)"
	}
	if newline := guessNewline(data); newline != "" {
		encoding += " (" + newline + ")"
	}
	return encoding + "\n", nil
}

func guessNewline(data []byte) string {
	newline := ""
	for i := 0; i < len(data); i++ {
		var current string
		switch data[i] {
		case '\r':
			current = "CR"
			if i+1 < len(data) && data[i+1] == '\n' {
				current = "CRLF"
				i++
			}
		case '\n':
			current = "LF"
		default:
			continue
		}
		if newline != "" && newline != current {
			return "MIXED NL"
		}
		newline = current
	}
	return newline
}

func guessenc(fileName string) error {
	_, err := guess(fileName)
	return err
}

func main() {
	if err := runCLI(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "go-nkf:", err)
		os.Exit(1)
	}
}
