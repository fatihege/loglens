package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/fatihege/loglens/internal/source"
)

func openInput(path string) (io.ReadCloser, error) {
	if path == "" || path == "-" {
		si, err := os.Stdin.Stat()
		if err != nil {
			return nil, fmt.Errorf("gather stat for stdin: %w", err)
		}

		if si.Mode()&os.ModeCharDevice != 0 {
			return nil, ErrTerminalInput
		}

		return source.Wrap(os.Stdin)
	}

	return source.Open(path)
}

func run() int {
	topFlag := flag.Int("top", 0, "display top n endpoints")

	flag.Parse()

	if *topFlag > 0 {
		fmt.Printf("top %d\n", *topFlag)
	}

	path := flag.Arg(0)
	rc, err := openInput(path)

	if err != nil {
		fmt.Fprintln(os.Stderr, err) // returned errors already has context

		if errors.Is(err, source.ErrPathIsDir) || errors.Is(err, ErrTerminalInput) {
			return 2
		}

		return 1
	}

	defer func() { _ = rc.Close() }()

	s := bufio.NewScanner(rc)
	lines := 0

	for s.Scan() {
		lines++
	}

	if err := s.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "reading tokens: %v", err)
		return 1
	}

	fmt.Println("read lines", lines)

	return 0
}

func main() {
	os.Exit(run())
}
