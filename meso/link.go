package meso

import (
	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/go-gmns/gmns/types"
	"github.com/LdDl/go-gmns/movement"
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

	segmentIdx int

	isConnection bool

	/* Movement information */
	movementID                    gmns.MovementID
	movementCompositeType         movement.MovementCompositeType // Inherited from movement
	movementMesoLinkIncome        gmns.LinkID
	movementMesoLinkOutcome       gmns.LinkID
	movementIncomeLaneStartSeqID  int
	movementOutcomeLaneStartSeqID int

	/* Inherited from paret data parameters */
	controlType       types.ControlType // Inherited from macroscopic node
	linkType          types.LinkType    // Inherited either from macroscopic link or from first incoming incident edge in macroscopic node
	freeSpeed         float64           // Inherited either from macroscopic link or from first incoming incident edge in macroscopic node
	capacity          int               // Inherited either from macroscopic link or from first incoming incident edge in macroscopic node
	allowedAgentTypes []types.AgentType // Inherited either from macroscopic link or from first incoming incident edge in macroscopic node
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
		controlType:                   types.CONTROL_TYPE_NOT_SIGNAL,
		linkType:                      types.LINK_UNDEFINED,
		freeSpeed:                     0.0,
		capacity:                      0,
		allowedAgentTypes:             []types.AgentType{},
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

func WithSegmentIdx(segmentIdx int) func(*Link) {
	return func(link *Link) {
		link.segmentIdx = segmentIdx
	}
}

func WithMovement(movementID gmns.MovementID) func(*Link) {
	return func(link *Link) {
		link.movementID = movementID
	}
}

func WithMovementCompositeType(movementCompositeType movement.MovementCompositeType) func(*Link) {
	return func(link *Link) {
		link.movementCompositeType = movementCompositeType
	}
}

func WithMovementLinkIncome(mesoLinkID gmns.LinkID) func(*Link) {
	return func(link *Link) {
		link.movementMesoLinkIncome = mesoLinkID
	}
}

func WithMovementLinkOutcome(mesoLinkID gmns.LinkID) func(*Link) {
	return func(link *Link) {
		link.movementMesoLinkOutcome = mesoLinkID
	}
}

func WithMovementIncomeLaneStartSeqID(startIncomeLaneSeqID int) func(*Link) {
	return func(link *Link) {
		link.movementIncomeLaneStartSeqID = startIncomeLaneSeqID
	}
}

func WithMovementOutcomeLaneStartSeqID(startOutcomeLaneSeqID int) func(*Link) {
	return func(link *Link) {
		link.movementOutcomeLaneStartSeqID = startOutcomeLaneSeqID
	}
}

func WithConnection(isConnection bool) func(*Link) {
	return func(link *Link) {
		link.isConnection = isConnection
	}
}

func WithControlType(controlType types.ControlType) func(*Link) {
	return func(link *Link) {
		link.controlType = controlType
	}
}

func WithLinkType(linkType types.LinkType) func(*Link) {
	return func(link *Link) {
		link.linkType = linkType
	}
}

func WithFreeSpeed(freeSpeed float64) func(*Link) {
	return func(link *Link) {
		link.freeSpeed = freeSpeed
	}
}

func WithCapacity(capacity int) func(*Link) {
	return func(link *Link) {
		link.capacity = capacity
	}
}

// WithAllowedAgentTypes sets allowed agent types for the link. Warning: it does copy argument
func WithAllowedAgentTypes(allowedAgentTypes []types.AgentType) func(*Link) {
	return func(link *Link) {
		link.allowedAgentTypes = make([]types.AgentType, len(allowedAgentTypes))
		copy(link.allowedAgentTypes, allowedAgentTypes)
	}
}

func (link *Link) SourceNodeID() gmns.NodeID {
	return link.sourceNodeID
}

func (link *Link) TargetNodeID() gmns.NodeID {
	return link.targetNodeID
}

func (link *Link) MacroLinkID() gmns.LinkID {
	return link.macroLinkID
}

func (link *Link) MacroNodeID() gmns.NodeID {
	return link.macroNodeID
}

func (link *Link) SegmentIdx() int {
	return link.segmentIdx
}

func (link *Link) IsConnection() bool {
	return link.isConnection
}

func (link *Link) LanesNum() int {
	return link.lanesNum
}

func (link *Link) LanesChange() [2]int {
	return link.lanesChange
}

// GeomEuclidean returns underlying euclidean geometry. Warning: returning object is a slice of Points.
func (link *Link) GeomEuclidean() orb.LineString {
	return link.geomEuclidean
}

// Geom returns underlying geometry. Warning: returning object is a slice of Points.
func (link *Link) Geom() orb.LineString {
	return link.geom
}

// LengthMeters returns geometry length in meters
func (link *Link) LengthMeters() float64 {
	return link.lengthMeters
}

// LinkType returns link type
func (link *Link) LinkType() types.LinkType {
	return link.linkType
}

// FreeSpeed returns free flow speed
func (link *Link) FreeSpeed() float64 {
	return link.freeSpeed
}

// Capacity returns max capacity
func (link *Link) Capacity() int {
	return link.capacity
}

// ControlType returns control type
func (link *Link) ControlType() types.ControlType {
	return link.controlType
}

// AllowedAgentTypes returns set of allowed agent types. Warning: returning object is a slice.
func (link *Link) AllowedAgentTypes() []types.AgentType {
	return link.allowedAgentTypes
}

// Movement returns attached movement ID
func (link *Link) Movement() gmns.MovementID {
	return link.movementID
}

// SetSourceNode sets source mesoscopic node
func (link *Link) SetSourceNode(sourceNodeID gmns.NodeID) {
	link.sourceNodeID = sourceNodeID
}

// SetTargetNode sets target mesoscopic node
func (link *Link) SetTargetNode(targetNodeID gmns.NodeID) {
	link.targetNodeID = targetNodeID
}
