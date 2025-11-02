package entities

//nolint:unused //will be fixed on the next time
type Report struct {
	kind      string
	format    string
	data      []byte
	recipient *User
}

func (r *Report) Type() string {
	return r.kind
}

func (r *Report) Data() []byte {
	return r.data
}
