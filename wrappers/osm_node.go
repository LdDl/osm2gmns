package wrappers

import (
	"github.com/LdDl/osm2gmns/types"
	"github.com/paulmach/osm"
)

type NodeOSM struct {
	InnerNode   osm.Node
	Name        string
	OsmData     NodeOSMInfo
	ID          osm.NodeID
	UseCount    int
	ControlType types.ControlType
	IsCrossing  bool
}

type NodeOSMInfo struct {
	Highway string
}

func NewNodeOSMFrom(node *osm.Node) *NodeOSM {
	nameText := node.Tags.Find("name")
	highwayText := node.Tags.Find("highway")
	controlType := types.CONTROL_TYPE_NOT_SIGNAL
	if highwayText == "traffic_signals" {
		controlType = types.CONTROL_TYPE_IS_SIGNAL
	}
	preparedNode := NodeOSM{
		Name:        nameText,
		InnerNode:   *node,
		ID:          node.ID,
		UseCount:    0,
		IsCrossing:  false,
		ControlType: controlType,
		OsmData: NodeOSMInfo{
			Highway: highwayText,
		},
	}
	return &preparedNode
}
