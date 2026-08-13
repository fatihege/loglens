package main

import (
	"fmt"

	"github.com/fatihege/loglens/internal/source"
)

func main() {
	reader := source.NewReader("./internal/source/reader.go")
	s, err := reader.Read()
	if err != nil {
		panic(err)
	}

	fmt.Print(s)
}
