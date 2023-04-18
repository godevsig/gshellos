package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

func main() {
	fmt.Println(os.Args)
	cnt := 30
	if len(os.Args) >= 2 {
		if i, err := strconv.Atoi(os.Args[1]); err == nil {
			cnt = i
		}
	}
	fmt.Printf("sleeping %d seconds\n", cnt)
	time.Sleep(time.Duration(cnt) * time.Second)
	fmt.Println("wakeup")
}
