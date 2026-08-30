package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"time"

	"github.com/fatihege/loglens/internal/lines"
	"github.com/fatihege/loglens/internal/parse"
	"github.com/fatihege/loglens/internal/source"
)

const MalformedErrorCap = 10

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
	start := time.Now()
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

	read, request, malformed, unrecognized, skipped := 0, 0, 0, 0, 0

	iter := lines.New(rc, filename, 64*1024)
	var p parse.Parser

	statuses := make(map[int]int)

	for {
		line, err := iter.Next()
		if errors.Is(err, io.EOF) {
			break
		} else if errors.Is(err, lines.ErrTooLong) {
			malformed++
			read++
			if malformed <= MalformedErrorCap {
				fmt.Fprintf(stderr, "%s:%d: %v\n", iter.Name(), iter.Num(), err)
			}
			continue
		} else if err != nil {
			fmt.Fprintf(stderr, "%s: %v\n", iter.Name(), err)
			return 1
		}

		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		if p == nil {
			if bytes.HasPrefix(line, []byte("{")) {
				p = parse.NewJSON()
			} else {
				p = parse.NewNginx()
			}
		}

		e, err := p.Parse(line)
		switch {
		case errors.Is(err, parse.ErrUnrecognized):
			unrecognized++
		case err != nil:
			malformed++
			if malformed <= MalformedErrorCap {
				fmt.Fprintf(stderr, "%s:%d: %v\n", iter.Name(), iter.Num(), err)
			}
		case !e.IsRequest():
			skipped++
		default:
			request++

			if e.Has(parse.FieldStatus) {
				statuses[e.Status]++
			}
		}

		read++
	}

	if malformed > MalformedErrorCap {
		fmt.Fprintf(
			stderr,
			"warning: %d malformed lines (%.2f%%); %d shown\n",
			malformed,
			float64(malformed*100)/float64(read),
			MalformedErrorCap,
		)
	}

	type pair struct {
		Key   int
		Value int
	}

	pairs := make([]pair, 0, len(statuses))

	for k, v := range statuses {
		pairs = append(pairs, pair{Key: k, Value: v})
	}

	slices.SortFunc(pairs, func(a, b pair) int {
		if a.Value != b.Value {
			return b.Value - a.Value
		}

		return a.Key - b.Key
	})

	if len(pairs) > 0 {
		fmt.Fprintln(stdout, "\nstatuses")

		for _, p := range pairs {
			fmt.Fprintf(stdout, "%d %d\n", p.Key, p.Value)
		}
	}

	fmt.Fprintf(stdout, `
  read lines %d
    requests %d
non-requests %d
unrecognized %d
   malformed %d

`, read, request, skipped, unrecognized, malformed)

	if p != nil {

		fmap := p.Fieldmap()
		fseen := p.FieldSeen()
		ferrs := p.FieldErrors()

		for _, k := range slices.Sorted(maps.Keys(ferrs)) {
			fmt.Fprintf(
				stderr,
				"warning: %s (from %q) unparseable on %d of %d lines that had it (%.2f%%)\n",
				k.String(),
				fmap[k],
				ferrs[k],
				fseen[k],
				float64(ferrs[k]*100)/float64(fseen[k]),
			)
		}
	}

	fmt.Fprintf(stdout, "\ntook %s to finish\n", time.Since(start).Round(10*time.Microsecond))
	return 0
}

func main() {
	var input io.Reader

	if fi, err := os.Stdin.Stat(); err == nil && fi.Mode()&os.ModeCharDevice == 0 {
		input = os.Stdin
	}

	os.Exit(run(os.Args[1:], input, os.Stdout, os.Stderr))
}
