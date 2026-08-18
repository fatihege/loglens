package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/fatihege/loglens/internal/lines"
	"github.com/fatihege/loglens/internal/source"
)

func openInput(path string, stdin io.Reader) (io.ReadCloser, string, error) {
	if path == "" || path == "-" {
		if stdin == nil {
			// nil means no stdin
			return nil, "", ErrNoInput
		}

		rc, err := source.Wrap(stdin)
		return rc, "stdin", err
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
	rc, filename, err := openInput(path, stdin)

	if err != nil {
		fmt.Fprintln(stderr, err) // returned errors already has context

		if errors.Is(err, source.ErrPathIsDir) || errors.Is(err, ErrNoInput) {
			return 2
		}

		return 1
	}

	defer func() { _ = rc.Close() }()

	count := 0
	malformed := 0

	iter := lines.New(rc, filename, 64*1024)

	for {
		_, err := iter.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if errors.Is(err, lines.ErrTooLong) {
			malformed++
			fmt.Fprintf(stderr, "%s:%d: %v\n", iter.Name(), iter.Num(), err)
			continue
		}
		if err != nil {
			fmt.Fprintf(stderr, "%s: %v", iter.Name(), err)
			return 1
		}
		count++
	}

	fmt.Fprintln(stdout, "read lines", count)
	fmt.Fprintln(stdout, "malformed", malformed)

	return 0
}

func main() {
	var input io.Reader = os.Stdin
	fi, err := os.Stdin.Stat()

	if err != nil {
		fmt.Fprintf(os.Stderr, "gather stats for stdin: %v\n", err)
		input = nil
	} else if fi.Mode()&os.ModeCharDevice != 0 {
		input = nil
	}

	os.Exit(run(os.Args[1:], input, os.Stdout, os.Stderr))
}
