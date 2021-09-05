package main

import (
	"fmt"
	"os"

	gs "github.com/godevsig/gshellos"
)

func main() {
	err := gs.ShellMain()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
