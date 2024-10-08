package macro

import (
	"fmt"
	"math"

	"github.com/LdDl/osm2gmns/geomath"
	"github.com/LdDl/osm2gmns/gmns"
	"github.com/LdDl/osm2gmns/types"
	"github.com/LdDl/osm2gmns/wrappers"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geo"
	"github.com/paulmach/osm"
)

type DirectionType uint16

const (
	DIRECTION_FORWARD = DirectionType(iota + 1)
	DIRECTION_BACKWARD
)

type Link struct {
	name               string
	geom               orb.LineString
	geomEuclidean      orb.LineString
	lengthMeters       float64
	freeSpeed          float64
	maxSpeed           float64
	capacity           int
	ID                 gmns.LinkID
	osmWayID           osm.WayID
	linkClass          types.LinkClass
	linkType           types.LinkType
	linkConnectionType types.LinkConnectionType
	controlType        types.ControlType
	allowedAgentTypes  []types.AgentType
	sourceNodeID       gmns.NodeID
	targetNodeID       gmns.NodeID

	sourceOsmNodeID osm.NodeID
	targetOsmNodeID osm.NodeID

	wasBidirectional bool

	lanesNum int
	/* For Mesoscopic and Microscopic */
	mesolinks              []gmns.LinkID
	lanesInfo              LanesInfo
	lanesListCut           []int
	lanesChangeCut         [][2]int
	geomOffset             orb.LineString
	geomOffsetCut          []orb.LineString
	geomEuclideanOffset    orb.LineString
	geomEuclideanOffsetCut []orb.LineString
	lengthMetersOffset     float64

	downstreamShortCut bool
	upstreamShortCut   bool

	downstreamIsTarget bool
	upstreamIsTarget   bool

	upstreamCutLen   float64
	downstreamCutLen float64
}

func (link *Link) GetIncomingLanes() int {
	if len(link.lanesInfo.LanesList) == 0 {
		return 0
	}
	return link.lanesInfo.LanesList[0]
}

func (link *Link) GetOutcomingLanes() int {
	idx := len(link.lanesInfo.LanesList) - 1
	if idx < 0 {
		return -1
	}
	return link.lanesInfo.LanesList[idx]
}

func (link *Link) GetOutcomingLaneIndices() []int {
	lanesInfo := link.lanesInfo
	idx := len(lanesInfo.LanesChange) - 1
	if idx < 0 {
		fmt.Printf("[WARNING]: Macroscopic link %d has no lanes change", link.ID)
		return make([]int, 0)
	}
	return laneIndices(link.lanesNum, lanesInfo.LanesChange[idx][0], lanesInfo.LanesChange[idx][1])
}

func NewLinkFrom(id gmns.LinkID, sourceNodeID, targetNodeID gmns.NodeID, sourceOSMNodeID, targetOSMNodeID osm.NodeID, direction DirectionType, way *wrappers.WayOSM, segmentNodes []*wrappers.NodeOSM) *Link {
	freeSpeed := -1.0
	maxSpeed := -1.0
	capacity := -1

	if way.Capacity < 0 {
		capacity = types.NewCapacityDefault(way.LinkType)
	}
	if way.FreeSpeed < 0 {
		if way.Tags.MaxSpeed >= 0 {
			freeSpeed = way.Tags.MaxSpeed
		} else {
			freeSpeed = types.NewSpeedDefault(way.LinkType)
			maxSpeed = freeSpeed
		}
	}

	link := Link{
		name:               way.Tags.Name,
		freeSpeed:          freeSpeed,
		maxSpeed:           maxSpeed,
		capacity:           capacity,
		ID:                 id,
		osmWayID:           way.ID,
		linkClass:          way.LinkClass,
		linkType:           way.LinkType,
		linkConnectionType: way.LinkConnectionType,
		sourceNodeID:       sourceNodeID,
		targetNodeID:       targetNodeID,
		sourceOsmNodeID:    sourceOSMNodeID,
		targetOsmNodeID:    targetOSMNodeID,
		controlType:        types.CONTROL_TYPE_NOT_SIGNAL,
		allowedAgentTypes:  make([]types.AgentType, len(way.AllowedAgentTypes)),
	}
	copy(link.allowedAgentTypes, way.AllowedAgentTypes)

	if !way.IsOneWay {
		link.wasBidirectional = true
	}
	if way.IsOneWay {
		link.lanesNum = way.Tags.Lanes
	} else {
		switch direction {
		case DIRECTION_FORWARD:
			if way.Tags.LanesForward > 0 {
				link.lanesNum = way.Tags.LanesForward
			} else if way.Tags.Lanes > 0 {
				link.lanesNum = int(math.Ceil(float64(way.Tags.Lanes) / 2.0))
			} else {
				link.lanesNum = way.Tags.Lanes
			}
		case DIRECTION_BACKWARD:
			if way.Tags.LanesBackward >= 0 {
				link.lanesNum = way.Tags.LanesBackward
			} else if way.Tags.Lanes >= 0 {
				link.lanesNum = int(math.Ceil(float64(way.Tags.Lanes) / 2.0))
			} else {
				link.lanesNum = way.Tags.Lanes
			}
		default:
			panic("Should not happen!")
		}
	}
	if link.lanesNum <= 0 {
		link.lanesNum = types.NewLanesDefault(link.linkType)
	}

	// Walk all segment nodes except the first and the last one to detect links under traffic light control
	for i := 1; i < len(segmentNodes)-1; i++ {
		node := segmentNodes[i]
		if node.ControlType == types.CONTROL_TYPE_IS_SIGNAL {
			link.controlType = node.ControlType
		}
	}

	// Prepare geometry
	link.geom = make(orb.LineString, 0, len(segmentNodes))
	switch direction {
	case DIRECTION_FORWARD:
		for _, node := range segmentNodes {
			pt := orb.Point{node.InnerNode.Lon, node.InnerNode.Lat}
			link.geom = append(link.geom, pt)
		}
	case DIRECTION_BACKWARD:
		for i := len(segmentNodes) - 1; i >= 0; i-- {
			node := segmentNodes[i]
			pt := orb.Point{node.InnerNode.Lon, node.InnerNode.Lat}
			link.geom = append(link.geom, pt)
		}
	default:
		panic("Should not happen!")
	}
	link.lengthMeters = geo.LengthHaversine(link.geom)
	link.geomEuclidean = geomath.LineToEuclidean(link.geom)

	// Prepare lanes information
	link.lanesInfo = NewLanesInfo(&link)
	return &link
}
