package types

type LinkConnectionType uint16

const (
	// Plain way
	NOT_A_LINK = LinkConnectionType(iota)
	// Connection between two roads
	IS_LINK
)

func (iotaIdx LinkConnectionType) String() string {
	return [...]string{"no", "yes"}[iotaIdx]
}
