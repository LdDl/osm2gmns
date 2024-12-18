package osmmacro

import (
	"math"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/go-gmns/gmns/types"
	"github.com/LdDl/go-gmns/macro"
	"github.com/LdDl/go-gmns/utils/geomath"
	"github.com/LdDl/osm2gmns/wrappers"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geo"
	"github.com/paulmach/osm"
)

func NewLinkFrom(id gmns.LinkID, sourceNodeID, targetNodeID gmns.NodeID, sourceOSMNodeID, targetOSMNodeID osm.NodeID, direction macro.DirectionType, way *wrappers.WayOSM, segmentNodes []*wrappers.NodeOSM) *macro.Link {
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

	link := macro.NewLinkFrom(
		id, sourceNodeID, targetNodeID,
		macro.WithLinkName(way.Tags.Name),
		macro.WithFreeSpeed(freeSpeed),
		macro.WithMaxSpeed(maxSpeed),
		macro.WithCapacity(capacity),
		macro.WithOSMWayID(way.ID),
		macro.WithSourceOSMNodeID(sourceOSMNodeID),
		macro.WithTargetOSMNodeID(targetOSMNodeID),
		macro.WithLinkClass(way.LinkClass),
		macro.WithLinkType(way.LinkType),
		macro.WithLinkConnectionType(way.LinkConnectionType),
		macro.WithLinkControlType(types.CONTROL_TYPE_NOT_SIGNAL),
		macro.WithAllowedAgentTypes(way.AllowedAgentTypes),
	)

	if !way.IsOneWay {
		macro.WithBidirectionalSource(true)(link)
	}
	if way.IsOneWay {
		macro.WithLanesNum(way.Tags.Lanes)(link)
	} else {
		switch direction {
		case macro.DIRECTION_FORWARD:
			if way.Tags.LanesForward > 0 {
				macro.WithLanesNum(way.Tags.LanesForward)(link)
			} else if way.Tags.Lanes > 0 {
				macro.WithLanesNum(int(math.Ceil(float64(way.Tags.Lanes) / 2.0)))(link)
			} else {
				macro.WithLanesNum(way.Tags.Lanes)(link)
			}
		case macro.DIRECTION_BACKWARD:
			if way.Tags.LanesBackward >= 0 {
				macro.WithLanesNum(way.Tags.LanesBackward)(link)
			} else if way.Tags.Lanes >= 0 {
				macro.WithLanesNum(int(math.Ceil(float64(way.Tags.Lanes) / 2.0)))(link)
			} else {
				macro.WithLanesNum(way.Tags.Lanes)(link)
			}
		default:
			panic("Should not happen!")
		}
	}
	if link.LanesNum() <= 0 {
		macro.WithLanesNum(types.NewLanesDefault(link.LinkType()))(link)
	}

	// Walk all segment nodes except the first and the last one to detect links under traffic light control
	for i := 1; i < len(segmentNodes)-1; i++ {
		node := segmentNodes[i]
		if node.ControlType == types.CONTROL_TYPE_IS_SIGNAL {
			macro.WithLinkControlType(node.ControlType)(link)
		}
	}

	// Prepare geometry
	linkGeom := make(orb.LineString, 0, len(segmentNodes))
	switch direction {
	case macro.DIRECTION_FORWARD:
		for _, node := range segmentNodes {
			pt := orb.Point{node.InnerNode.Lon, node.InnerNode.Lat}
			linkGeom = append(linkGeom, pt)
		}
	case macro.DIRECTION_BACKWARD:
		for i := len(segmentNodes) - 1; i >= 0; i-- {
			node := segmentNodes[i]
			pt := orb.Point{node.InnerNode.Lon, node.InnerNode.Lat}
			linkGeom = append(linkGeom, pt)
		}
	default:
		panic("Should not happen!")
	}
	macro.WithLineGeom(linkGeom)(link)
	macro.WithLengthMeters(geo.LengthHaversine(linkGeom))(link)
	macro.WithLineGeomEuclidean(geomath.LineToEuclidean(linkGeom))(link)

	// Prepare lanes information
	macro.WithLanesInfo(macro.NewLanesInfo(link))(link)
	return link
}
