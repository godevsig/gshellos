package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	as "github.com/godevsig/adaptiveservice"
)

func usage() string {
	return os.Args[0] + ` <list|tag msgNameList|show tokenList>

list:
    list traceable message type names
tag <msgNameList>:
    Tag the message types in msgNameList to be traced and return tracing tokens
    msgNameList is a comma-separated list containing the names of the message types
show <tokenList>:
    Display the tracing results specified by a list of tokens
    tokenList is a comma-separated list containing tracing tokens
`
}

func main() {
	args := os.Args
	if len(args) < 2 {
		fmt.Println("wrong usage, see --help")
		return
	}
	switch args[1] {
	case "-h", "--help":
		fmt.Println(usage())
	case "list":
		types := as.GetKnownMessageTypes()
		sort.Slice(types, func(i, j int) bool {
			return types[i] < types[j]
		})
		for i, name := range types {
			fmt.Println(i, name)
		}
	case "tag":
		if len(args) != 3 {
			fmt.Println("wrong usage, see --help")
			return
		}
		traceList := args[2]
		fields := strings.Split(traceList, ",")
		for _, name := range fields {
			token, err := as.TraceMsgByName(name)
			if err != nil {
				fmt.Println(err)
				continue
			}
			fmt.Printf("Tracing <%s> with token %s\n\n", name, token)
		}
	case "show":
		if len(args) != 3 {
			fmt.Println("wrong usage, see --help")
			return
		}
		tokenList := args[2]
		tokens := strings.Split(tokenList, ",")
		for _, token := range tokens {
			fmt.Printf("\nTraced records with token: %s\n", token)
			msgs, err := as.ReadTracedMsg(token)
			if err != nil {
				fmt.Println(err)
				continue
			}
			fmt.Println(msgs)
		}
	}
}
