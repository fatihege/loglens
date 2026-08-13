package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/fatihege/loglens/internal/source"
)

func main() {
	topFlag := flag.Int("top", 0, "display top n endpoints")

	flag.Parse()

	if *topFlag > 0 {
		fmt.Printf("top %d\n", *topFlag)
	}

	filePath := flag.Arg(0)
	if filePath == "" {
		fmt.Fprintln(os.Stderr, "no file path provided")
		os.Exit(1)
	}

	rc, err := source.Open(filePath)
	if err != nil {
		if errors.Is(err, source.ErrPathIsDir) {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "error opening source: %v\n", err)
		}
		os.Exit(1)
	}
	defer rc.Close()
}
