package osm2gmns

type ControlType uint16

const (
	CONTROL_TYPE_UNDEFINED = ControlType(iota)
	CONTROL_TYPE_NOT_SIGNAL
	CONTROL_TYPE_IS_SIGNAL
)

func (iotaIdx ControlType) String() string {
	return [...]string{"undefined", "common", "signal"}[iotaIdx]
}
