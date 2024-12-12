package meso

import (
	"github.com/LdDl/osm2gmns/gmns"
	"github.com/LdDl/osm2gmns/types"
	"github.com/paulmach/orb"
)

type Node struct {
	ID gmns.NodeID

	geom          orb.Point
	geomEuclidean orb.Point

	macroNodeID gmns.NodeID
	macroLinkID gmns.LinkID

	macroZoneID      gmns.NodeID        // Should be inherited from the macroscopic node
	activityLinkType types.LinkType     // Should be inherited from the macroscopic node
	boundaryType     types.BoundaryType // Should be evaluated from macroscopic node and macroscopic link

	incomingLinks  map[gmns.LinkID]struct{}
	outcomingLinks map[gmns.LinkID]struct{}
}

func NewNodeFrom(id gmns.NodeID, options ...func(*Node)) *Node {
	newNode := &Node{
		ID:               id,
		macroNodeID:      -1,
		macroLinkID:      -1,
		macroZoneID:      -1,
		activityLinkType: types.LINK_UNDEFINED,
		boundaryType:     types.BOUNDARY_NONE,
		incomingLinks:    make(map[gmns.LinkID]struct{}),
		outcomingLinks:   make(map[gmns.LinkID]struct{}),
	}
	for _, option := range options {
		option(newNode)
	}
	return newNode
}

func WithPointMacroNode(macroNodeID gmns.NodeID) func(*Node) {
	return func(node *Node) {
		node.macroNodeID = macroNodeID
	}
}

func WithPointMacroLink(macroLinkID gmns.LinkID) func(*Node) {
	return func(node *Node) {
		node.macroLinkID = macroLinkID
	}
}

func WithMacroZone(macroZoneID gmns.NodeID) func(*Node) {
	return func(node *Node) {
		node.macroZoneID = macroZoneID
	}
}

func WithActivityLinkType(activityLinkType types.LinkType) func(*Node) {
	return func(node *Node) {
		node.activityLinkType = activityLinkType
	}
}

func WithBoundaryType(boundaryType types.BoundaryType) func(*Node) {
	return func(node *Node) {
		node.boundaryType = boundaryType
	}
}

func WithPointGeom(geom orb.Point) func(*Node) {
	return func(node *Node) {
		node.geom = geom
	}
}

func WithPointEuclideanGeom(geomEuclidean orb.Point) func(*Node) {
	return func(node *Node) {
		node.geomEuclidean = geomEuclidean
	}
}

func WithIncomingLinks(linksIDs ...gmns.LinkID) func(*Node) {
	return func(node *Node) {
		for i := range linksIDs {
			node.incomingLinks[linksIDs[i]] = struct{}{}
		}
	}
}

func WithOutcomingLinks(linksIDs ...gmns.LinkID) func(*Node) {
	return func(node *Node) {
		for i := range linksIDs {
			node.outcomingLinks[linksIDs[i]] = struct{}{}
		}
	}
}

func (node *Node) AddIncomingLinks(linksIDs ...gmns.LinkID) {
	for i := range linksIDs {
		node.incomingLinks[linksIDs[i]] = struct{}{}
	}
}

func (node *Node) AddOutcomingLinks(linksIDs ...gmns.LinkID) {
	for i := range linksIDs {
		node.outcomingLinks[linksIDs[i]] = struct{}{}
	}
}

// GeomEuclidean returns underlying euclidean geometry
func (node *Node) GeomEuclidean() orb.Point {
	return node.geomEuclidean
}

// Geom returns underlying geometry
func (node *Node) Geom() orb.Point {
	return node.geom
}

// MacroNodeID returns parent macroscopic node identifier. Outputs "-1" of there is no information about parent macroscopic node.
func (node *Node) MacroNodeID() gmns.NodeID {
	return node.macroNodeID
}

// MacroLinkID returns parent macroscopic link identifier. Outputs "-1" of there is no information about parent macroscopic link.
func (node *Node) MacroLinkID() gmns.LinkID {
	return node.macroLinkID
}

// MacroZoneID returns ID of macroscopic zone. Outputs "-1" of there is no information about parent macroscopic zone ID.
func (node *Node) MacroZoneID() gmns.NodeID {
	return node.macroZoneID
}

// ActivityLinkType returns type of activity link. Outputs LINK_UNDEFINED if there is no information.
func (node *Node) ActivityLinkType() types.LinkType {
	return node.activityLinkType
}

// BoundaryType returns boundary type. Outputs BOUNDARY_NONE if there is no information.
func (node *Node) BoundaryType() types.BoundaryType {
	return node.boundaryType
}
