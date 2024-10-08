package macro

import (
	"github.com/LdDl/osm2gmns/geomath"
	"github.com/LdDl/osm2gmns/gmns"
	"github.com/LdDl/osm2gmns/movement"
	"github.com/LdDl/osm2gmns/types"
	"github.com/LdDl/osm2gmns/wrappers"
	"github.com/paulmach/orb"
	"github.com/paulmach/osm"
)

type Node struct {
	incomingLinks    []gmns.LinkID
	outcomingLinks   []gmns.LinkID
	name             string
	osmHighway       string
	ID               gmns.NodeID
	osmNodeID        osm.NodeID
	intersectionID   int
	zoneID           gmns.NodeID
	poiID            PoiID
	controlType      types.ControlType
	boundaryType     types.BoundaryType
	activityType     types.ActivityType
	activityLinkType types.LinkType
	geom             orb.Point
	geomEuclidean    orb.Point

	/* Mesoscopic */
	movements        []*movement.Movement
	movementIsNeeded bool

	/* Not used */
	isCentroid bool
}

func NewNodeFrom(id gmns.NodeID, node *wrappers.NodeOSM) *Node {
	newNode := Node{
		incomingLinks:    make([]gmns.LinkID, 0),
		outcomingLinks:   make([]gmns.LinkID, 0),
		activityType:     types.ACTIVITY_NONE,
		name:             node.Name,
		osmHighway:       node.OsmData.Highway,
		ID:               id,
		osmNodeID:        node.ID,
		intersectionID:   -1,
		zoneID:           -1,
		poiID:            -1,
		controlType:      node.ControlType,
		boundaryType:     types.BOUNDARY_NONE,
		geom:             node.InnerNode.Point(),
		movementIsNeeded: true, // Consider all nodes as intersections by default
	}
	newNode.geomEuclidean = geomath.PointToEuclidean(newNode.geom)
	return &newNode
}
