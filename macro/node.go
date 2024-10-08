package macro

import (
	"github.com/LdDl/osm2gmns/geomath"
	"github.com/LdDl/osm2gmns/gmns"
	"github.com/LdDl/osm2gmns/movement"
	"github.com/LdDl/osm2gmns/types"
	"github.com/LdDl/osm2gmns/wrappers"
	"github.com/paulmach/orb"
	"github.com/paulmach/osm"
	"github.com/pkg/errors"
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

func (node *Node) FindMovements(links map[gmns.LinkID]*Link) ([]movement.Movement, error) {
	movements := []movement.Movement{}

	income := len(node.incomingLinks)
	outcome := len(node.outcomingLinks)
	if income == 0 || outcome == 0 {
		return movements, nil
	}

	if outcome == 1 {
		// Merge
		incomingLinksList := []*Link{}
		outcomingLinkID := node.outcomingLinks[0]
		outcomingLink, ok := links[outcomingLinkID]
		if !ok {
			return nil, errors.Wrapf(ErrLinkNotFound, "Outcoming Link ID: %d", outcomingLinkID)
		}
		for i := range node.incomingLinks {
			incomingLinkID := node.incomingLinks[i]
			incomingLink, ok := links[incomingLinkID]
			if !ok {
				return nil, errors.Wrapf(ErrLinkNotFound, "Incoming Link ID: %d", incomingLinkID)
			}
			if incomingLink.sourceNodeID != outcomingLink.targetNodeID { // Ignore all reverse directions
				incomingLinksList = append(incomingLinksList, incomingLink)
			}
		}
		if len(incomingLinksList) == 0 {
			return movements, nil
		}

		connections := getSpansConnections(outcomingLink, incomingLinksList)
		incomingLaneIndices := outcomingLink.GetOutcomingLaneIndices()
		for i := range incomingLinksList {
			incomingLink := incomingLinksList[i]
			incomeLaneIndexStart := connections[i][0].first
			incomeLaneIndexEnd := connections[i][0].second
			outcomeLaneIndexStart := connections[i][1].first
			outcomeLaneIndexEnd := connections[i][1].second
			lanesNum := incomeLaneIndexEnd - incomeLaneIndexStart + 1

			outcomingLaneIndices := incomingLink.GetOutcomingLaneIndices()
			mvmtTextID, mvmtType := movement.FindMovementType(incomingLink.geomEuclidean, outcomingLink.geomEuclidean)
			mvmtGeom := movement.FindMovementGeom(incomingLink.geom, outcomingLink.geom)
			mvmt := movement.NewMovement(
				node.ID, incomingLink.ID, outcomingLinkID, mvmtTextID, mvmtType, mvmtGeom,
				movement.WithOSMNode(node.osmNodeID),
				movement.WithSourceOSMNode(incomingLink.sourceOsmNodeID),
				movement.WithTargetOSMNode(outcomingLink.targetOsmNodeID),
				movement.WithControlType(node.controlType),
				movement.WithAllowedAgentTypes(incomingLink.allowedAgentTypes),
				movement.WithLanesNum(lanesNum),
				movement.WithIncomeLane(outcomingLaneIndices[incomeLaneIndexStart], outcomingLaneIndices[incomeLaneIndexEnd]),
				movement.WithIncomeLaneSequence(incomeLaneIndexStart, incomeLaneIndexEnd),
				movement.WithOutcomeLane(incomingLaneIndices[outcomeLaneIndexStart], incomingLaneIndices[outcomeLaneIndexEnd]),
				movement.WithOutcomeLaneSequence(outcomeLaneIndexStart, outcomeLaneIndexEnd),
			)
			movements = append(movements, mvmt)
		}
	} else {
		// Diverge
		// Intersections
		for i := range node.incomingLinks {
			incomingLinkID := node.incomingLinks[i]
			incomingLink, ok := links[incomingLinkID]
			if !ok {
				return nil, errors.Wrapf(ErrLinkNotFound, "Intersection incoming Link ID: %d", incomingLinkID)
			}
			outcomingLinksList := []*Link{}
			for j := range node.outcomingLinks {
				outcomingLinkID := node.outcomingLinks[j]
				outcomingLink, ok := links[outcomingLinkID]
				if !ok {
					return nil, errors.Wrapf(ErrLinkNotFound, "Intersection outcoming Link ID: %d", outcomingLinkID)
				}
				if incomingLink.sourceNodeID != outcomingLink.targetNodeID { // Ignore all reverse directions
					outcomingLinksList = append(outcomingLinksList, outcomingLink)
				}
			}
			if len(outcomingLinksList) == 0 {
				return movements, nil
			}
			// @todo
			connections := getIntersectionsConnections(incomingLink, outcomingLinksList)
			outcomingLaneIndices := incomingLink.GetOutcomingLaneIndices()

			for i := range outcomingLinksList {
				outcomingLink := outcomingLinksList[i]
				incomeLaneIndexStart := connections[i][0].first
				incomeLaneIndexEnd := connections[i][0].second
				outcomeLaneIndexStart := connections[i][1].first
				outcomeLaneIndexEnd := connections[i][1].second
				lanesNum := incomeLaneIndexEnd - incomeLaneIndexStart + 1

				incomingLaneIndices := outcomingLink.GetOutcomingLaneIndices()
				mvmtTextID, mvmtType := movement.FindMovementType(incomingLink.geomEuclidean, outcomingLink.geomEuclidean)
				mvmtGeom := movement.FindMovementGeom(incomingLink.geom, outcomingLink.geom)
				mvmt := movement.NewMovement(
					node.ID, incomingLinkID, outcomingLink.ID, mvmtTextID, mvmtType, mvmtGeom,
					movement.WithOSMNode(node.osmNodeID),
					movement.WithSourceOSMNode(incomingLink.sourceOsmNodeID),
					movement.WithTargetOSMNode(outcomingLink.targetOsmNodeID),
					movement.WithControlType(node.controlType),
					movement.WithAllowedAgentTypes(incomingLink.allowedAgentTypes),
					movement.WithLanesNum(lanesNum),
					movement.WithIncomeLane(outcomingLaneIndices[incomeLaneIndexStart], outcomingLaneIndices[incomeLaneIndexEnd]),
					movement.WithIncomeLaneSequence(incomeLaneIndexStart, incomeLaneIndexEnd),
					movement.WithOutcomeLane(incomingLaneIndices[outcomeLaneIndexStart], incomingLaneIndices[outcomeLaneIndexEnd]),
					movement.WithOutcomeLaneSequence(outcomeLaneIndexStart, outcomeLaneIndexEnd),
				)
				movements = append(movements, mvmt)
			}
		}
	}

	return movements, nil
}
