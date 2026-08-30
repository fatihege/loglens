package parse

type Parser interface {
	Fieldmap() map[FieldMask]string
	FieldErrors() map[FieldMask]int
	FieldSeen() map[FieldMask]int
	Parse(line []byte) (Entry, error)
}
