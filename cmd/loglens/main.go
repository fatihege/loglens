package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/fatihege/loglens/internal/source"
)

func run() int {
	topFlag := flag.Int("top", 0, "display top n endpoints")

	flag.Parse()

	if *topFlag > 0 {
		fmt.Printf("top %d\n", *topFlag)
	}

	filePath := flag.Arg(0)
	if filePath == "" {
		fmt.Fprintln(os.Stderr, "no file path provided")
		return 2
	}

	rc, err := source.Open(filePath)
	if err != nil {
		switch {
		case errors.Is(err, source.ErrPathIsDir) || errors.Is(err, source.ErrTerminalInput):
			fmt.Fprintln(os.Stderr, err)
			return 2
		default:
			fmt.Fprintf(os.Stderr, "error opening source: %v\n", err)
			return 1
		}

	}

	defer rc.Close()

	return 0
}

func main() {
	os.Exit(run())
}
