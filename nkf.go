package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"

	"golang.org/x/net/html/charset"
	"golang.org/x/text/transform"
)

func guess(fileName string) (string, error) {
	cmd := exec.Command("nkf", "--guess", fileName)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	str := string(output)
	for _, substr := range []string{"UTF-8", "ASCII"} {
		if strings.Contains(str, substr) {
			return str, nil
		}
	}
	return str, fmt.Errorf("Unsupported %s encoding", str)
}

func guessenc(fileName string) error {
	enc, _ := charset.Lookup("utf-8")
	if enc == nil {
		return fmt.Errorf("Unsupported utf-8 charset")
	}
	f, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	ic := transform.NewWriter(&buf, enc.NewDecoder())
	if _, err := ic.Write(f); err != nil {
		return err
	}
	if err := ic.Close(); err != nil {
		return err
	}

	return nil
}

func main() {

}
