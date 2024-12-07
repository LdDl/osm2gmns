package meso

import (
	"github.com/LdDl/osm2gmns/gmns"
	"github.com/LdDl/osm2gmns/movement"
	"github.com/paulmach/orb"
)

type Link struct {
	ID gmns.LinkID

	geom          orb.LineString
	geomEuclidean orb.LineString
	lanesNum      int
	lanesChange   [2]int

	lengthMeters float64

	sourceNodeID gmns.NodeID
	targetNodeID gmns.NodeID

	macroNodeID gmns.NodeID
	macroLinkID gmns.LinkID

	isConnection bool

	/* Movement information */
	movementID                    movement.MovementID
	movementCompositeType         movement.MovementCompositeType // Inherited from movement
	movementMesoLinkIncome        gmns.LinkID
	movementMesoLinkOutcome       gmns.LinkID
	movementIncomeLaneStartSeqID  int
	movementOutcomeLaneStartSeqID int
}

func NewLinkFrom(id gmns.LinkID, sourceNodeID, targetNodeID gmns.NodeID, options ...func(*Link)) *Link {
	newLink := &Link{
		ID:           id,
		sourceNodeID: sourceNodeID,
		targetNodeID: targetNodeID,
		isConnection: false,
		// Default movement
		movementCompositeType:         movement.MOVEMENT_UNDEFINED,
		movementMesoLinkIncome:        gmns.LinkID(-1),
		movementMesoLinkOutcome:       gmns.LinkID(-1),
		movementIncomeLaneStartSeqID:  -1,
		movementOutcomeLaneStartSeqID: -1,
	}
	for _, option := range options {
		option(newLink)
	}
	return newLink
}

func WithLineMacroNode(macroNodeID gmns.NodeID) func(*Link) {
	return func(link *Link) {
		link.macroNodeID = macroNodeID
	}
}

func WithLineMacroLink(macroLinkID gmns.LinkID) func(*Link) {
	return func(link *Link) {
		link.macroLinkID = macroLinkID
	}
}

func WithLineGeom(geom orb.LineString) func(*Link) {
	return func(link *Link) {
		link.geom = geom
	}
}

func WithLineEuclideanGeom(geomEuclidean orb.LineString) func(*Link) {
	return func(link *Link) {
		link.geomEuclidean = geomEuclidean
	}
}

func WithLengthMeters(lengthMeters float64) func(*Link) {
	return func(link *Link) {
		link.lengthMeters = lengthMeters
	}
}

func WithLanesNum(lanesNum int) func(*Link) {
	return func(link *Link) {
		link.lanesNum = lanesNum
	}
}

func WithLanesChange(lanesChange [2]int) func(*Link) {
	return func(link *Link) {
		link.lanesChange = lanesChange
	}
}

func WithMovement(movementID movement.MovementID) func(*Link) {
	return func(link *Link) {
		link.movementID = movementID
	}
}

func Connection(isConnection bool) func(*Link) {
	return func(link *Link) {
		link.isConnection = isConnection
	}
}
