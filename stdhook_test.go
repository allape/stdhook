package stdhook

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

const Password = "123_567"

func TestHook(t *testing.T) {
	var output []byte
	config := &Config{
		Timeout:     5 * time.Second,
		TriggerWord: ":",
		onTrigger: func(channel int, content string) string {
			lines := strings.Split(content, "\n")
			lastLine := lines[len(lines)-1]
			if lastLine == "Please enter your password:" {
				return Password + "\n"
			}
			return ""
		},
		onOutput: func(channel int, content []byte) {
			output = append(output, content...)
		},
	}

	err := Hook(config, "bash", "./mock.sh")

	fmt.Println("final output:", string(output))

	if !strings.HasSuffix(strings.TrimSpace(string(output)), Password) {
		t.Error("Password not found in the output")
	}

	if err != nil {
		t.Error(err)
	}
}
