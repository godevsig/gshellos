package main

import (
	"fmt"
	"os"

	gs "github.com/godevsig/gshellos"
	_ "github.com/godevsig/gshellos/extension" // register all extensions
)

func main() {
	err := gs.ShellMain()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
