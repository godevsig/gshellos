package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	as "github.com/godevsig/adaptiveservice"
)

func usage() string {
	return os.Args[0] + ` <list|tag ...|untag ... |show ...|purge>

list:
    list traceable message type names
tag <msgNameList> [count <number>]:
    Tag the message types specified in <msgNameList> for tracing and return tracing tokens.
    <msgNameList> is a comma-separated list containing the names of the message types.
    Each token corresponds one tracing session associated with a message type.
    The tracing stops after sending <number> messages maching specified message type, generating
    a set of sequential tokens sharing the same prefix, in below form if number is 100:
    750768e4-f572-4c4e-9302-46d84c756361.0..99
    The default value for <number> is 1.
untag <all|msgNameList>:
    Untag all message types that have been tagged or the message types specified in <msgNameList>.
show <tokenList>:
    Display the tracing results specified by a list of tokens.
    <tokenList> is a comma-separated list containing tracing tokens.
purge:
    Read and remove all traced messages for all tracing sessions, including those triggered by others.
    CAUTION: this will trigger a force cleanup across all service nodes, resulting missing traced message
    records for other tracing sessions even on remote nodes.
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
		count := uint32(1)
		switch len(args) {
		case 5:
			if args[3] != "count" {
				fmt.Println("wrong usage, see --help")
				return
			}
			num, err := strconv.ParseUint(args[4], 10, 32)
			if err != nil {
				fmt.Println("parse count error:", err)
				return
			}
			count = uint32(num)
			fallthrough
		case 3:
			fields := strings.Split(args[2], ",")
			for _, name := range fields {
				token, err := as.TraceMsgByNameWithCount(name, count)
				if err != nil {
					fmt.Println(err)
					continue
				}
				fmt.Printf("Tracing <%s> with token %s\n\n", name, token)
			}
		default:
			fmt.Println("wrong usage, see --help")
		}
	case "untag":
		if len(args) != 3 {
			fmt.Println("wrong usage, see --help")
			return
		}
		if args[2] == "all" {
			as.UnTraceMsgAll()
		} else {
			fields := strings.Split(args[2], ",")
			for _, name := range fields {
				_, err := as.TraceMsgByNameWithCount(name, 0)
				if err != nil {
					fmt.Println(err)
					continue
				}
				fmt.Printf("Tracing <%s> stopped\n", name)
			}
		}
	case "show":
		if len(args) != 3 {
			fmt.Println("wrong usage, see --help")
			return
		}
		for _, mt := range strings.Split(args[2], ",") {
			count := 1
			strs := strings.Split(mt, "..")
			if len(strs) == 2 {
				num, err := strconv.ParseUint(strs[1], 10, 32)
				if err != nil {
					fmt.Println("parse count error:", err)
					continue
				}
				count = int(num) + 1
			}
			prefix := strings.TrimSuffix(strs[0], ".0")
			for i := 0; i < count; i++ {
				token := fmt.Sprintf("%s.%d", prefix, i)
				fmt.Printf("\nTraced records with token: %s\n", token)
				msgs, err := as.ReadTracedMsg(token)
				if err != nil {
					fmt.Println(err)
					continue
				}
				fmt.Println(msgs)
			}
		}
	case "purge":
		msgs, err := as.ReadAllTracedMsg()
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(msgs)
	}
}
