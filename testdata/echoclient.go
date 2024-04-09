package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	as "github.com/godevsig/adaptiveservice"
	"github.com/godevsig/grepo/echo"
)

var running = true

func doEcho(conn as.Connection) {
	var rep echo.Reply
	req := echo.Request{
		Msg: "ni hao",
		Num: 0,
	}
	for i := 0; running && i < 3; i++ {
		req.Num += 100
		if err := conn.SendRecv(req, &rep); err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("%v ==> %v, %s\n", req, rep.Request, rep.Signature)
		//time.Sleep(time.Second)
	}

	var wg sync.WaitGroup
	for i := 0; running && i < 3; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			stream := conn.NewStream()
			req := echo.Request{
				Msg: "ni hao",
				Num: 100 * int32(i),
			}
			var rep echo.Reply
			for i := 0; running && i < 5; i++ {
				req.Num += 10
				if err := stream.SendRecv(req, &rep); err != nil {
					fmt.Println(err)
					return
				}
				if req.Num+1 != rep.Num {
					panic("wrong number")
				}
				fmt.Printf("%v ==> %v, %s\n", req, rep.Request, rep.Signature)
				//time.Sleep(time.Second)
			}
		}(i)
	}
	wg.Wait()
}

func doWhoelse(conn as.Connection) {
	go func() {
		eventStream := conn.NewStream()
		if err := eventStream.SendRecv(echo.SubWhoElseEvent{}, nil); err != nil {
			fmt.Println(err)
			return
		}
		for running {
			var addr string
			if err := eventStream.Recv(&addr); err != nil {
				fmt.Println(err)
				return
			}
			fmt.Printf("event: new client %s\n", addr)
		}
	}()

	for i := 0; running && i < 200; i++ {
		var whoelse string
		if err := conn.SendRecv(echo.WhoElse{}, &whoelse); err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("clients: %s\n", whoelse)
		time.Sleep(3 * time.Second)
	}
	return
}

// Start starts the app
func Start(args []string) (err error) {
	flags := flag.NewFlagSet("", flag.ContinueOnError)
	flags.SetOutput(os.Stdout)

	cmd := flags.String("cmd", "echo", "echo or whoelse")

	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			err = nil
		}
		return err
	}

	c := as.NewClient().SetDiscoverTimeout(3)
	conn := <-c.Discover(echo.Publisher, echo.ServiceEcho)
	if conn == nil {
		return as.ErrServiceNotFound(echo.Publisher, echo.ServiceEcho)
	}
	defer conn.Close()

	switch *cmd {
	case "echo":
		doEcho(conn)
	case "whoelse":
		doWhoelse(conn)
	default:
		return errors.New("unknown cmd")
	}

	return nil
}

// Stop stops the app
func Stop() {
	fmt.Println("echo client stopping...")
	running = false
}

func main() {
	if err := Start(os.Args[1:]); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
