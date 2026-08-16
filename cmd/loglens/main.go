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

func openInput(path string, stdin io.Reader) (io.ReadCloser, error) {
	if path == "" || path == "-" {
		if stdin == nil {
			// nil means no stdin
			return nil, ErrTerminalInput
		}

		return source.Wrap(stdin)
	}

	return source.Open(path)
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fset := flag.NewFlagSet("loglens", flag.ContinueOnError)

	fset.SetOutput(stderr)

	topFlag := fset.Int("top", 0, "display top n endpoints")

	if err := fset.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}

		return 2
	}

	if *topFlag > 0 {
		fmt.Fprintf(stdout, "top %d\n", *topFlag)
	}

	path := fset.Arg(0)
	rc, err := openInput(path, stdin)

	if err != nil {
		fmt.Fprintln(stderr, err) // returned errors already has context

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
		fmt.Fprintf(stderr, "reading tokens: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, "read lines", lines)

	return 0
}

func main() {
	var input io.Reader = os.Stdin
	fi, err := os.Stdin.Stat()

	if err != nil {
		fmt.Fprintf(os.Stderr, "gather stats for stdin: %v\n", err)
		os.Exit(1)
	}

	if fi.Mode()&os.ModeCharDevice != 0 {
		input = nil
	}

	os.Exit(run(os.Args[1:], input, os.Stdout, os.Stderr))
}
