package parse

import "testing"

func TestEntry(t *testing.T) {
	wantField := true
	var e Entry

	if e.Mask != 0 {
		t.Errorf("e.Mask = %v, want %v", e.Mask, 0)
	}

	e.Level = "TEST"
	e.Mark(FieldLevel)

	if got := e.Has(FieldLevel); got != wantField {
		t.Errorf("e.Has(Level) = %v, want %v", got, wantField)
	}

	if e.Mask == 0 {
		t.Errorf("e.Mask = %v, want non-zero", e.Mask)
	}
}
