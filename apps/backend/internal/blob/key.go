package blob

type Key string

type KeyMode int

const (
	KeyModeStable KeyMode = iota
	KeyModeDerived
)

func (k Key) IsNil() bool {
	return k == ""
}
