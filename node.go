package osm2gmns

import (
	"github.com/LdDl/osm2gmns/types"
	"github.com/paulmach/osm"
)

type Node struct {
	node        osm.Node
	name        string
	osmData     NodeOSMInfo
	ID          osm.NodeID
	useCount    int
	controlType types.ControlType
	isCrossing  bool
}

type NodeOSMInfo struct {
	highway string
}
