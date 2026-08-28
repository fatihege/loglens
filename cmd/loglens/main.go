package main

import (
	"bytes"
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

	count, parsed, malformed := 0, 0, 0

	iter := lines.New(rc, filename, 64*1024)
	j := parse.NewJSON()

	statuses := make(map[int]int)

	for {
		line, err := iter.Next()
		if errors.Is(err, io.EOF) {
			break
		} else if errors.Is(err, lines.ErrTooLong) {
			malformed++
			fmt.Fprintf(stderr, "%s:%d: %v\n", iter.Name(), iter.Num(), err)
			continue
		} else if err != nil {
			fmt.Fprintf(stderr, "%s: %v\n", iter.Name(), err)
			return 1
		}

		if e, err := j.Parse(line); bytes.HasPrefix(line, []byte("{")) && err != nil {
			fmt.Fprintf(stderr, "%s:%d: %v\n", iter.Name(), iter.Num(), err)
			malformed++
		} else if e.Has(parse.FieldStatus) {
			statuses[e.Status]++
			parsed++
		} else {
			malformed++
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
		if pairs[i].Value != pairs[j].Value {
			return pairs[i].Value > pairs[j].Value
		}

		return pairs[i].Key < pairs[j].Key
	})

	fmt.Fprintln(stdout, "\nstatuses")
	for _, p := range pairs {
		fmt.Fprintf(stdout, "%d\t%d\n", p.Key, p.Value)
	}

	fmt.Fprintln(stdout, "\nread lines", count)
	fmt.Fprintln(stdout, "parsed", parsed)
	fmt.Fprintln(stdout, "malformed", malformed)
	fmt.Fprintln(stdout, "")

	fm := j.Fieldmap()
	for f, e := range j.FieldErrors() {
		fmt.Fprintf(
			stderr,
			"warning: %q unparseable on %v of %v lines (%.2f%%)\n",
			fm[f],
			e,
			count,
			float32(e*100)/float32(count),
		)
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
