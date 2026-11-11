package enums

type Access string

const (
	AccessRead  Access = "read"
	AccessWrite Access = "write"
)

func (a Access) Valid() bool {
	return a == AccessRead || a == AccessWrite
}
