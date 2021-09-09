package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("sleeping")
	time.Sleep(30 * time.Second)
	fmt.Println("wakeup")
}
