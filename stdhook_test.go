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
		Timeout:               5 * time.Second,
		TriggerWord:           ":",
		OnlyTriggerOnLastLine: true,
		OnTrigger: func(channel int, content string) string {
			if content == "Please enter your password:" {
				return Password + "\n"
			}
			return ""
		},
		OnOutput: func(channel int, content []byte) {
			output = append(output, content...)
		},
	}

	err := Hook(config, "bash", "./mock.sh")

	if err != nil {
		t.Error(err)
	}

	fmt.Println("final output:", string(output))

	if !strings.HasSuffix(strings.TrimSpace(string(output)), Password) {
		t.Error("Password not found in the output")
	}

	output = []byte{}
	config.Timeout = 1 * time.Second
	config.OnlyTriggerOnLastLine = false
	err = Hook(config, "bash", "./mock.sh")
	if err == nil {
		t.Error("Expecting error")
	}
}
