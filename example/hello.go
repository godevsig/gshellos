package main

import (
	"fmt"
	"os"
)

type testA struct {
	A int
	B string
}

func main() {
	fmt.Println("Hello, playground")
	fmt.Println(os.Args)
	fmt.Printf("%#v\n", testA{})
}
