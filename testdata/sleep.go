package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

var stopChan = make(chan struct{})

// Stop stops the app
func Stop() {
	fmt.Println("stopping...")
	close(stopChan)
}

func main() {
	fmt.Println(os.Args)
	cnt := 30
	if len(os.Args) >= 2 {
		if i, err := strconv.Atoi(os.Args[1]); err == nil {
			cnt = i
		}
	}
	fmt.Printf("sleeping %d seconds\n", cnt)
	go func() {
		time.Sleep(time.Duration(cnt) * time.Second)
		stopChan <- struct{}{}
	}()

	_, ok := <-stopChan
	if ok {
		fmt.Println("wakeup")
	} else {
		fmt.Println("canceled")
	}
}
