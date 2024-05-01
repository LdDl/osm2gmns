package osm2gmns

import (
	"github.com/LdDl/osm2gmns/types"
	"github.com/LdDl/osm2gmns/wrappers"
	"github.com/paulmach/osm"
)

type WayType uint16

const (
	WAY_TYPE_UNDEFINED = WayType(iota)
	WAY_TYPE_HIGHWAY
	WAY_TYPE_RAILWAY
	WAY_TYPE_AEROWAY
)

// type Way struct {
// 	Nodes  osm.WayNodes
// 	TagMap osm.Tags
// 	ID     osm.WayID
// 	Oneway bool
// }

type WayOSM struct {
	tags wrappers.WayTags
	// geom               orb.LineString
	allowedAgentTypes   []types.AgentType
	Nodes               []osm.NodeID
	segments            [][]osm.NodeID
	osmSourceNodeID     osm.NodeID
	capacity            int
	ID                  osm.WayID
	wayType             WayType
	isHighwayNegligible bool
	freeSpeed           float64
	osmTargetNodeID     osm.NodeID
	linkConnectionType  types.LinkConnectionType
	linkType            types.LinkType
	linkClass           types.LinkClass
	isPureCycle         bool
	isCycle             bool
	// POI information (optional)
	wayPOI *WayPOIProps
	// The rest of params
	isArea   bool
	isOneWay bool
}

func (way *WayOSM) IsPureCycle() bool {
	return way.isPureCycle
}
