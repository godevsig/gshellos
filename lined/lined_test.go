package lined

import (
	"errors"
	"fmt"
	"io"
	"testing"
)

func TestLined(t *testing.T) {
	led := NewEditor(Cfg{
		Prompt: ">> ",
	})
	defer led.Close()

	for {
		line, err := led.Readline()
		if errors.Is(err, io.EOF) {
			break
		}
		if len(line) != 0 {
			fmt.Println(line)
		}
	}
}
