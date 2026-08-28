package parse

import "testing"

func TestEntry(t *testing.T) {
	var e Entry

	if e.Mask != 0 {
		t.Errorf("e.Mask = %v, want %v", e.Mask, 0)
	}

	if e.IsRequest() {
		t.Errorf("e.IsRequest() = %v, want %v", e.IsRequest(), false)
	}

	e.Level = "TEST"
	e.Mark(FieldLevel)

	if !e.Has(FieldLevel) {
		t.Errorf("e.Has(Level) = %v, want %v", e.Has(FieldLevel), true)
	}

	if e.Mask == 0 {
		t.Errorf("e.Mask = %v, want non-zero", e.Mask)
	}

	e.Method = "POST"
	e.Mark(FieldMethod)

	if !e.IsRequest() {
		t.Errorf("e.IsRequest() = %v, want %v", e.IsRequest(), true)
	}
}
