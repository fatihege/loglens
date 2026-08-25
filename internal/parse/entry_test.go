package parse

import "testing"

func TestEntryMark(t *testing.T) {
	want := true
	var e Entry

	e.Level = "TEST"
	e.Mark(FieldLevel)

	if got := e.Has(FieldLevel); got != want {
		t.Errorf("e.Has(Level) = %v, want %v", got, want)
	}
}
