package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/fatihege/loglens/internal/lines"
	"github.com/fatihege/loglens/internal/parse"
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
	exceed := 0

	iter := lines.New(rc, filename, 64*1024)
	j := parse.NewJSON()

	statuses := make(map[int]int)

	for {
		line, err := iter.Next()
		if errors.Is(err, io.EOF) {
			break
		} else if errors.Is(err, lines.ErrTooLong) {
			exceed++
			fmt.Fprintf(stderr, "%s:%d: %v\n", iter.Name(), iter.Num(), err)
			continue
		} else if err != nil {
			fmt.Fprintf(stderr, "%s: %v\n", iter.Name(), err)
			return 1
		}

		if e, err := j.Parse(line); err != nil {
			fmt.Fprintf(stderr, "%s:%d: %v\n", iter.Name(), iter.Num(), err)
		} else if e.Has(parse.FieldStatus) {
			statuses[e.Status]++
		}

		count++
	}

	type pair struct {
		Key   int
		Value int
	}

	pairs := make([]pair, 0, len(statuses))

	for k, v := range statuses {
		pairs = append(pairs, pair{Key: k, Value: v})
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Value > pairs[j].Value
	})

	fmt.Fprintln(stdout, "read lines", count)
	fmt.Fprintln(stdout, "exceeded", exceed)

	fmt.Fprintln(stdout, "\nstatuses")
	for _, p := range pairs {
		fmt.Fprintf(stdout, "%d\t%d\n", p.Key, p.Value)
	}

	return 0
}

func main() {
	var input io.Reader

	if fi, err := os.Stdin.Stat(); err == nil && fi.Mode()&os.ModeCharDevice == 0 {
		input = os.Stdin
	}

	os.Exit(run(os.Args[1:], input, os.Stdout, os.Stderr))
}
