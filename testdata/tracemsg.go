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
    List traceable message type names
tag <msgName> [count <number>] [filters <filterList>]:
    Tag the message types specified in <msgNameList> for tracing and return tracing tokens.
    <number>: The tracing stops after sending <number> messages maching specified message type,
    generating a set of sequential tokens sharing the same prefix, in below form if number is 100:
    750768e4-f572-4c4e-9302-46d84c756361.0..99
    The default value for <number> is 1.
    <filterList>: comma-separated list of filters in the form field1=pattern1,field2=pattern2 ...
    The default value for <filterList> is nil.
untag <all|msgNameList>:
    Untag all message types that have been tagged or the message types specified in <msgNameList>.
    <msgNameList> is a comma-separated list containing the names of the message types.
show <tokenList>:
    Display the tracing results specified by a list of tokens.
    Note: Tracing results are read cleared, so 2nd show with the same token will be empty.
    <tokenList> is a comma-separated list containing tracing tokens returned by tag subcommand.
purge:
    Read and remove all traced messages for all tracing sessions, including those triggered by others.
    CAUTION: this will trigger a force cleanup across all service nodes, resulting missing traced message.
    records for other tracing sessions even on remote nodes.
`
}

func do() (err error) {
	args := os.Args
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("wrong usage, see --help")
		}
	}()

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
		msgName := args[2]
		count := uint32(1)
		var filters []string

		args = args[3:]
		for len(args) > 0 {
			switch args[0] {
			case "count":
				num, err := strconv.ParseUint(args[1], 10, 32)
				if err != nil {
					return fmt.Errorf("parse count error: %v", err)
				}
				count = uint32(num)
			case "filters":
				strs := strings.Split(args[1], ",")
				for _, str := range strs {
					if len(strings.Split(str, "=")) != 2 {
						return fmt.Errorf("%s filter format error", str)
					}
					filters = append(filters, str)
				}
			}
			args = args[2:]
		}

		token, err := as.TraceMsgByNameWithFilters(msgName, count, filters)
		if err != nil {
			return err
		}
		fmt.Printf("Tracing <%s> with token %s\n\n", msgName, token)
	case "untag":
		if args[2] == "all" {
			as.UnTraceMsgAll()
		} else {
			strs := strings.Split(args[2], ",")
			for _, name := range strs {
				_, err := as.TraceMsgByName(name, 0)
				if err != nil {
					fmt.Println(err)
					continue
				}
				fmt.Printf("Tracing <%s> stopped\n", name)
			}
		}
	case "show":
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
	return nil
}

func main() {
	if err := do(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
