package osmmacro

import (
	"fmt"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/go-gmns/gmns/types"
	"github.com/LdDl/go-gmns/macro"
	"github.com/LdDl/osm2gmns/wrappers"
	"github.com/paulmach/osm"
	"github.com/pkg/errors"
)

func NewNetFromOSM(ways []*wrappers.WayOSM, nodesSet map[osm.NodeID]*wrappers.NodeOSM) (*macro.Net, error) {

	lastLinkID := gmns.LinkID(0)
	lastNodeID := gmns.NodeID(0)

	observed := make(map[osm.NodeID]gmns.NodeID)
	nodes := make(map[gmns.NodeID]*macro.Node)
	links := make(map[gmns.LinkID]*macro.Link)

	for i := range ways {
		way := ways[i]
		if way.IsPureCycle {
			continue
		}
		segments, err := prepareSegments(way, nodesSet)
		if err != nil {
			return nil, errors.Wrapf(err, "can't prepare segments for way: %d", way.ID)
		}
		for j := range segments {
			segment := segments[j]
			if len(segment) < 2 {
				continue
			}
			var currentSourceNodeID gmns.NodeID
			var currentTargetNodeID gmns.NodeID

			/* Create nodes */
			sourceNodeID := segment[0]
			if nID, ok := observed[sourceNodeID]; !ok {
				sourceNode, ok := nodesSet[sourceNodeID]
				if !ok {
					return nil, fmt.Errorf("no such source node '%d'. Way ID: '%d'", sourceNodeID, way.ID)
				}
				nodes[lastNodeID] = NewNodeFrom(lastNodeID, sourceNode)
				observed[sourceNodeID] = lastNodeID
				currentSourceNodeID = lastNodeID
				lastNodeID++
			} else {
				currentSourceNodeID = nID
			}
			targetNodeID := segment[len(segment)-1]
			if nID, ok := observed[targetNodeID]; !ok {
				targetNode, ok := nodesSet[targetNodeID]
				if !ok {
					return nil, fmt.Errorf("no such target node '%d'. Way ID: '%d'", targetNodeID, way.ID)
				}
				nodes[lastNodeID] = NewNodeFrom(lastNodeID, targetNode)
				observed[targetNodeID] = lastNodeID
				currentTargetNodeID = lastNodeID
				lastNodeID++
			} else {
				currentTargetNodeID = nID
			}

			/* Create links */
			nodesForSegment := make([]*wrappers.NodeOSM, len(segment))
			for i, nodeID := range segment {
				nodesForSegment[i] = nodesSet[nodeID]
			}
			links[lastLinkID] = NewLinkFrom(lastLinkID, currentSourceNodeID, currentTargetNodeID, nodes[currentSourceNodeID].OSMNode(), nodes[currentTargetNodeID].OSMNode(), macro.DIRECTION_FORWARD, way, nodesForSegment)
			macro.WithOutcomingLinks(lastLinkID)(nodes[currentSourceNodeID])
			macro.WithIncomingLinks(lastLinkID)(nodes[currentTargetNodeID])
			lastLinkID++
			if !way.IsOneWay {
				links[lastLinkID] = NewLinkFrom(lastLinkID, currentTargetNodeID, currentSourceNodeID, nodes[currentTargetNodeID].OSMNode(), nodes[currentSourceNodeID].OSMNode(), macro.DIRECTION_BACKWARD, way, nodesForSegment)
				macro.WithOutcomingLinks(lastLinkID)(nodes[currentTargetNodeID])
				macro.WithIncomingLinks(lastLinkID)(nodes[currentSourceNodeID])
				lastLinkID++
			}
		}
	}

	net := &macro.Net{Nodes: nodes, Links: links}
	genBoundaryAndActivityType(net)
	return net, nil
}

func prepareSegments(way *wrappers.WayOSM, nodesSet map[osm.NodeID]*wrappers.NodeOSM) (segments [][]osm.NodeID, err error) {
	nodesNum := len(way.Nodes)
	lastNodeIdx := 0
	idx := 0
	for {
		segmentNodes := []osm.NodeID{way.Nodes[lastNodeIdx]}
		for idx = lastNodeIdx + 1; idx < nodesNum; idx++ {
			nextNodeID := way.Nodes[idx]
			nextNode, ok := nodesSet[nextNodeID]
			if !ok {
				return segments, fmt.Errorf("no such node: %d", nextNodeID)
			}
			segmentNodes = append(segmentNodes, nextNodeID)
			if nextNode.IsCrossing {
				lastNodeIdx = idx
				break
			}
		}
		segments = append(segments, segmentNodes)
		if idx == nodesNum-1 {
			break
		}
	}
	return segments, nil
}

// genBoundaryAndActivityType updated BoundaryType, ActivityType, ActivityLinkType for nodes
// In case when counters for acitivites are equal prioritization will be used
func genBoundaryAndActivityType(macroNet *macro.Net) error {
	nodesLinkTypesCounters := make(map[gmns.NodeID]map[types.LinkType]int)
	for i := range macroNet.Links {
		link := macroNet.Links[i]
		sourceNodeID := link.SourceNode()
		if _, ok := macroNet.Nodes[sourceNodeID]; !ok {
			return fmt.Errorf("no source node with ID '%d'. Link ID: '%d'", sourceNodeID, link.ID)
		}
		if _, ok := nodesLinkTypesCounters[sourceNodeID]; !ok {
			nodesLinkTypesCounters[sourceNodeID] = make(map[types.LinkType]int)
		}
		linkType := link.LinkType()
		if _, ok := nodesLinkTypesCounters[sourceNodeID][linkType]; !ok {
			nodesLinkTypesCounters[sourceNodeID][linkType] = 1
		} else {
			nodesLinkTypesCounters[sourceNodeID][linkType]++
		}

		targetNodeID := link.TargetNode()
		if _, ok := macroNet.Nodes[targetNodeID]; !ok {
			return fmt.Errorf("no target node with ID '%d'. Link ID: '%d'", targetNodeID, link.ID)
		}
		if _, ok := nodesLinkTypesCounters[targetNodeID]; !ok {
			nodesLinkTypesCounters[targetNodeID] = make(map[types.LinkType]int)
		}
		if _, ok := nodesLinkTypesCounters[targetNodeID][linkType]; !ok {
			nodesLinkTypesCounters[targetNodeID][linkType] = 1
		} else {
			nodesLinkTypesCounters[targetNodeID][linkType]++
		}
	}

	for nodeID := range macroNet.Nodes {
		node := macroNet.Nodes[nodeID]
		if node.POI() > -1 {
			macro.WithActivityType(types.ACTIVITY_POI)(node)
			macro.WithActivityLinkType(types.LINK_UNDEFINED)(node)
		}
		if linkTypesCounters, ok := nodesLinkTypesCounters[nodeID]; ok {
			maxLinkTypes := []types.LinkType{types.LINK_UNDEFINED}
			maxLinkTypeCount := 0
			for linkType, counter := range linkTypesCounters {
				if counter > maxLinkTypeCount {
					maxLinkTypeCount = counter
					maxLinkTypes = append(maxLinkTypes, linkType)
				}
			}
			if maxLinkTypeCount > 0 {
				macro.WithActivityType(types.ACTIVITY_LINK)(node)
				// When there are several link types pick the one with highest rank
				macro.WithActivityLinkType(types.FindPriorLinkType(maxLinkTypes))(node)
			} else {
				macro.WithActivityType(types.ACTIVITY_NONE)(node)
				macro.WithActivityLinkType(types.LINK_UNDEFINED)(node)
			}
		}
	}

	for nodeID := range macroNet.Nodes {
		node := macroNet.Nodes[nodeID]
		macro.WithBoundaryType(types.BOUNDARY_NONE)(node)
		if node.ActivityType() == types.ACTIVITY_POI {
			continue
		}
		incomeLinks := node.IncomingLinks()
		outcomeLinks := node.OutcomingLinks()
		if len(outcomeLinks) == 0 {
			macro.WithBoundaryType(types.BOUNDARY_INCOME_ONLY)(node)
		} else if len(incomeLinks) == 0 {
			macro.WithBoundaryType(types.BOUNDARY_OUTCOME_ONLY)(node)
		} else if len(incomeLinks) == 1 && (len(outcomeLinks) == 1) {
			incomingLink, ok := macroNet.Links[incomeLinks[0]]
			if !ok {
				return fmt.Errorf("no incoming link with ID '%d'. Node ID: '%d'", incomeLinks[0], node.ID)
			}
			outcomingLink, ok := macroNet.Links[outcomeLinks[0]]
			if !ok {
				return fmt.Errorf("no incoming link with ID '%d'. Node ID: '%d'", outcomeLinks[0], node.ID)
			}
			if incomingLink.SourceNode() == outcomingLink.TargetNode() {
				macro.WithBoundaryType(types.BOUNDARY_INCOME_OUTCOME)(node)
			}
		}
	}
	for nodeID := range macroNet.Nodes {
		node := macroNet.Nodes[nodeID]
		if node.BoundaryType() == types.BOUNDARY_NONE {
			continue
		}
		macro.WithZoneID(node.ID)(node)
	}
	return nil
}
