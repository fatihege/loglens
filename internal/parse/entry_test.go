package parse

import "testing"

func TestEntry(t *testing.T) {
	wantField := true
	wantEmpty := true
	var e Entry

	if got := e.IsEmpty(); got != wantEmpty {
		t.Errorf("e.IsEmpty() = %v, want %v", got, wantEmpty)
	}

	e.Level = "TEST"
	e.Mark(FieldLevel)

	if got := e.Has(FieldLevel); got != wantField {
		t.Errorf("e.Has(Level) = %v, want %v", got, wantField)
	}

	wantEmpty = false
	if got := e.IsEmpty(); got != wantEmpty {
		t.Errorf("e.IsEmpty() = %v, want %v", got, wantEmpty)
	}
}
