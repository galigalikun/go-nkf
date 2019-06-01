package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"

	"golang.org/x/text/encoding/japanese"
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
	f, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	transform.NewWriter(buf, japanese.UTF.NewEncoder())
	_, err := ic.Write(body)
	if err != nil {
		return err
	}
	if err := ic.Close(); err != nil {
		return err
	}

	return nil
}

func main() {

}
