package main

import (
	"path/filepath"
	"testing"
)

func TestGuess(t *testing.T) {
	if _, err := guess(filepath.Join("testdata", "test.euc-jp.txt")); err == nil {
		t.Errorf("test.euc-jp.txt, UTF-8 or ASCII と判定されている??")
		return
	}
	if _, err := guess(filepath.Join("testdata", "test.shift-jis.txt")); err == nil {
		t.Errorf("test.shift-jis.txt UTF-8 or ASCII と判定されている??")
		return
	}
	if _, err := guess(filepath.Join("testdata", "test.utf-8.txt")); err != nil {
		t.Errorf("test.utf-8.txt, err = %s", err.Error())
		return
	}
	if err := guess(filepath.Join("testdata", "test.txt")); err != nil {
		t.Errorf("test.utf-8.txt, err = %s", err.Error())
		return
	}
}

func TestGuessEnc(t *testing.T) {
	if err := guessenv(filepath.Join("testdata", "test.euc-jp.txt")); err == nil {
		t.Errorf("test.euc-jp.txt, UTF-8 と判定されている??")
		return
	}
	if err := guessenv(filepath.Join("testdata", "test.shift-jis.txt")); err == nil {
		t.Errorf("test.shift-jis.txt UTF-8 と判定されている??")
		return
	}
	if err := guessenv(filepath.Join("testdata", "test.utf-8.txt")); err != nil {
		t.Errorf("test.utf-8.txt, err = %s", err.Error())
		return
	}
	if err := guessenv(filepath.Join("testdata", "test.txt")); err != nil {
		t.Errorf("test.utf-8.txt, err = %s", err.Error())
		return
	}
}
