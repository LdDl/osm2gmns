package osm2gmns

import (
	"github.com/paulmach/osm"
)

type Node struct {
	node        osm.Node
	name        string
	highway     string
	ID          osm.NodeID
	useCount    int
	controlType ControlType
	isCrossing  bool
}
