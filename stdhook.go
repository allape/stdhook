package stdhook

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"strings"
	"time"
)

type Config struct {
	Timeout               time.Duration                            // timeout for the command
	TriggerWord           string                                   // trigger word to trigger the onTrigger function
	OnlyTriggerOnLastLine bool                                     // only trigger on the last line of the output
	OnTrigger             func(channel int, content string) string // function to handle the trigger
	OnOutput              func(channel int, content []byte)        // function to handle the output
}

type Payload struct {
	Message []byte
	Channel int
}

func Hook(config *Config, cmd string, args ...string) error {
	if config.OnTrigger == nil {
		return errors.New("onTrigger is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()
	command := exec.CommandContext(ctx, cmd, args...)

	stdin, err := command.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		return err
	}

	var (
		done         = false
		outputChan   = make(chan *Payload)
		asyncErrChan = make(chan error)
	)

	defer func() {
		done = true
		close(outputChan)
		close(asyncErrChan)
	}()

	go func() {
		outputCache := [][]byte{
			{},
			{},
			{},
		}
		for {
			payload, ok := <-outputChan
			if !ok || done {
				break
			}

			if config.OnOutput != nil {
				config.OnOutput(payload.Channel, payload.Message)
			}
			//if payload.Channel == 1 {
			//	_, _ = fmt.Fprint(os.Stdout, payload.Message)
			//} else {
			//	_, _ = fmt.Fprint(os.Stderr, payload.Message)
			//}
			outputCache[payload.Channel] = append(outputCache[payload.Channel], payload.Message...)

			output := strings.TrimSpace(string(outputCache[payload.Channel]))
			if strings.HasSuffix(output, config.TriggerWord) {
				if config.OnlyTriggerOnLastLine {
					lines := strings.Split(output, "\n")
					output = strings.TrimSpace(lines[len(lines)-1])
				}
				input := config.OnTrigger(payload.Channel, output)
				if input != "" {
					outputCache[payload.Channel] = []byte{}
					_, err := stdin.Write([]byte(input))
					if err != nil {
						if !done {
							asyncErrChan <- err
						}
						break
					}
				}
			}
		}
	}()

	listenStdout := func(reader io.Reader, channelIndex int) {
		go func() {
			defer func() {
				// may emit "send data into a closed channel"
				_ = recover()
			}()
			eof := false
			buffer := make([]byte, 1024)
			for {
				n, err := reader.Read(buffer)
				if err != nil {
					if err == io.EOF {
						eof = true
					} else {
						if !done {
							asyncErrChan <- err
						}
						break
					}
				}

				buffer := buffer[:n]
				if done {
					break
				}
				outputChan <- &Payload{
					Message: buffer,
					Channel: channelIndex,
				}

				if eof || done {
					if !done {
						asyncErrChan <- nil
					}
					return
				}
			}
		}()
	}

	listenStdout(stdout, 1)
	listenStdout(stderr, 2)

	err = command.Start()
	if err != nil {
		return err
	}

	select {
	case err := <-asyncErrChan:
		if err != nil {
			return err
		}
	case <-ctx.Done():
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}

	return nil
}
