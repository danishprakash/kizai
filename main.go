package main

import (
	"fmt"
	"os"

	"github.com/danishprakash/kizai/command"
)

func main() {
	args := os.Args

	helpStr := `usage: kizai [options]
static site generator.
options:
    build                       builds the current project`

	if len(args) <= 1 {
		fmt.Println(helpStr)
		return
	}

	switch args[1] {
	case "build":
		command.Build()
	case "serve":
		command.Serve()
	default:
		fmt.Println(helpStr)
	}
}
