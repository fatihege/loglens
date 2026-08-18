package parse

import (
	"encoding/json"
	"fmt"
	"slices"
)

type JSON struct {
	keymap map[string]string
}

func NewJSON() *JSON {
	return &JSON{}
}

func (j *JSON) Do(line []byte) (Entry, error) {
	var entry Entry
	var rawEntry map[string]json.RawMessage
	err := json.Unmarshal(line, &rawEntry)
	if err != nil {
		return Entry{}, err
	}

	if j.keymap == nil {
		j.keymap = mapKeys(rawEntry)
	}

	fmt.Printf("%q\n", j.keymap)
	fmt.Printf("%q\n", rawEntry)

	entry, err = mapEntry(j.keymap, rawEntry)

	return entry, nil
}

func mapKeys(entry map[string]json.RawMessage) map[string]string {
	keymap := make(map[string]string, 0)

	for k := range entry {
		if match := mapKey(k); match != "" {
			keymap[match] = k
		}
	}

	return keymap
}

func mapKey(key string) string {
	for k, v := range EntryKeys {
		if slices.Contains(v, key) {
			return k
		}
	}

	return ""
}

func mapEntry(keymap map[string]string, src map[string]json.RawMessage) (Entry, error) {
	return Entry{}, nil
}
