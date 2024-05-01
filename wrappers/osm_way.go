package wrappers

import (
	"github.com/LdDl/osm2gmns/types"
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
	Tags WayTags
	// geom               orb.LineString
	AllowedAgentTypes   []types.AgentType
	Nodes               []osm.NodeID
	segments            [][]osm.NodeID
	capacity            int
	ID                  osm.WayID
	WayType             WayType
	IsHighwayNegligible bool
	freeSpeed           float64
	OsmSourceNodeID     osm.NodeID
	OsmTargetNodeID     osm.NodeID
	LinkConnectionType  types.LinkConnectionType
	LinkType            types.LinkType
	LinkClass           types.LinkClass
	IsPureCycle         bool
	IsCycle             bool
	// POI information (optional)
	WayPOI *WayPOIProps
	// The rest of params
	IsArea   bool
	IsOneWay bool
}

func NewWayOSMFrom(way *osm.Way) *WayOSM {
	// Call tags flattening to make further processing easier
	tags := NewWayTagsFrom(way)

	wayType := WAY_TYPE_UNDEFINED
	poiName := ""
	poiType := types.POI_TYPE_UNDEFINED
	if tags.IsHighway() {
		wayType = WAY_TYPE_HIGHWAY
		if tags.IsHighwayPOI() {
			poiName = tags.Highway
			poiType = types.POI_TYPE_HIGHWAY
		}
	} else if tags.IsRailway() {
		wayType = WAY_TYPE_RAILWAY
		if tags.IsRailwayPOI() {
			poiName = tags.Railway
			poiType = types.POI_TYPE_RAILWAY
		}
	} else if tags.IsAeroway() {
		wayType = WAY_TYPE_AEROWAY
		if tags.IsAerowayPOI() {
			poiName = tags.Aeroway
			poiType = types.POI_TYPE_AEROWAY
		}
	}

	preparedWay := &WayOSM{
		ID:                  way.ID,
		WayType:             wayType,
		IsHighwayNegligible: tags.IsHighwayNegligible(),
		Nodes:               make([]osm.NodeID, 0, len(way.Nodes)),
		Tags:                tags,
		freeSpeed:           -1.0,
		capacity:            -1.0,
		IsArea:              tags.Area != "" && tags.Area != "no",
		IsOneWay:            tags.Oneway,
	}
	if poiName != "" {
		preparedWay.WayPOI = &WayPOIProps{
			poiName: poiName,
			PoiType: poiType,
		}
	}
	for _, node := range way.Nodes {
		preparedWay.Nodes = append(preparedWay.Nodes, node.ID)
	}
	return preparedWay
}
