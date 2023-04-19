package main

import (
	"fmt"
	"os"

	"github.com/godevsig/grepo/lib/sys/log"
)

func main() {
	fmt.Println("Hello, playground")

	stream := log.NewStream("")
	defer stream.Close()
	stream.SetOutputter(os.Stdout)
	lg := stream.NewLogger("main", log.Linfo)
	defer lg.Close()
	lg.Debugln("this is debug line ")
	lg.Infoln("this is info line ")
	lg.Errorln("this is error line")
}
