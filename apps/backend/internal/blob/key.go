package blob

type Key string

func (k Key) IsNil() bool {
	return k == ""
}
