package main

import (
	"fmt"
	"os/exec"
	"strings"
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

func main() {

}
