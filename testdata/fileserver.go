package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
)

var lnr net.Listener

// Start starts the service
func Start(args []string) (err error) {
	flags := flag.NewFlagSet("", flag.ContinueOnError)
	flags.SetOutput(os.Stdout)
	dir := flags.String("dir", "", "directory to be served")
	port := flags.String("port", "8088", "port for http")
	if err = flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			err = nil
		}
		return err
	}

	if len(*dir) == 0 {
		return fmt.Errorf("no dir specified")
	}
	fmt.Printf("file server for %s @ :%s\n", *dir, *port)

	lnr, err = net.Listen("tcp", ":"+*port)
	if err != nil {
		return err
	}
	defer func() {
		lnr.Close()
		fmt.Println("file server stopped")
	}()

	fs := http.FileServer(http.Dir(*dir))
	fmt.Println("file server running...")
	if err := http.Serve(lnr, fs); err != nil {
		return err
	}

	return nil
}

// Stop stops the service
func Stop() {
	fmt.Println("file server stopping...")
	lnr.Close()
}

func main() {
	if err := Start(os.Args[1:]); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
