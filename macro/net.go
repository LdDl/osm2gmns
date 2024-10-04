package macro

import (
	"fmt"

	"github.com/LdDl/osm2gmns/types"
	"github.com/LdDl/osm2gmns/wrappers"
	"github.com/paulmach/osm"
	"github.com/pkg/errors"
)

var (
	ErrLinkNotFound = fmt.Errorf("Link not found")
)

type Net struct {
	Nodes map[NodeID]*Node
	Links map[LinkID]*Link
}

func NewNetFromOSM(ways []*wrappers.WayOSM, nodesSet map[osm.NodeID]*wrappers.NodeOSM) (*Net, error) {

	lastLinkID := LinkID(0)
	lastNodeID := NodeID(0)

	observed := make(map[osm.NodeID]NodeID)
	nodes := make(map[NodeID]*Node)
	links := make(map[LinkID]*Link)

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
			var currentSourceNodeID NodeID
			var currentTargetNodeID NodeID

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

			// @todo: Prepare nodes and links
			/* Create links */
			nodesForSegment := make([]*wrappers.NodeOSM, len(segment))
			for i, nodeID := range segment {
				nodesForSegment[i] = nodesSet[nodeID]
			}
			links[lastLinkID] = NewLinkFrom(lastLinkID, currentSourceNodeID, currentTargetNodeID, nodes[currentSourceNodeID].osmNodeID, nodes[currentTargetNodeID].osmNodeID, DIRECTION_FORWARD, way, nodesForSegment)
			nodes[currentSourceNodeID].outcomingLinks = append(nodes[currentSourceNodeID].outcomingLinks, lastLinkID)
			nodes[currentTargetNodeID].incomingLinks = append(nodes[currentTargetNodeID].incomingLinks, lastLinkID)
			lastLinkID++
			if !way.IsOneWay {
				links[lastLinkID] = NewLinkFrom(lastLinkID, currentTargetNodeID, currentSourceNodeID, nodes[currentTargetNodeID].osmNodeID, nodes[currentSourceNodeID].osmNodeID, DIRECTION_BACKWARD, way, nodesForSegment)
				nodes[currentTargetNodeID].outcomingLinks = append(nodes[currentTargetNodeID].outcomingLinks, lastLinkID)
				nodes[currentSourceNodeID].incomingLinks = append(nodes[currentSourceNodeID].incomingLinks, lastLinkID)
				lastLinkID++
			}
		}
	}

	net := &Net{Nodes: nodes, Links: links}
	net.genBoundaryAndActivityType()
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
func (net *Net) genBoundaryAndActivityType() error {
	nodesLinkTypesCounters := make(map[NodeID]map[types.LinkType]int)
	for i := range net.Links {
		link := net.Links[i]
		sourceNodeID := link.sourceNodeID
		if _, ok := net.Nodes[sourceNodeID]; !ok {
			return fmt.Errorf("no source node with ID '%d'. Link ID: '%d'", sourceNodeID, link.ID)
		}
		if _, ok := nodesLinkTypesCounters[sourceNodeID]; !ok {
			nodesLinkTypesCounters[sourceNodeID] = make(map[types.LinkType]int)
		}
		if _, ok := nodesLinkTypesCounters[sourceNodeID][link.linkType]; !ok {
			nodesLinkTypesCounters[sourceNodeID][link.linkType] = 1
		} else {
			nodesLinkTypesCounters[sourceNodeID][link.linkType]++
		}

		targetNodeID := link.targetNodeID
		if _, ok := net.Nodes[targetNodeID]; !ok {
			return fmt.Errorf("no target node with ID '%d'. Link ID: '%d'", targetNodeID, link.ID)
		}
		if _, ok := nodesLinkTypesCounters[targetNodeID]; !ok {
			nodesLinkTypesCounters[targetNodeID] = make(map[types.LinkType]int)
		}
		if _, ok := nodesLinkTypesCounters[targetNodeID][link.linkType]; !ok {
			nodesLinkTypesCounters[targetNodeID][link.linkType] = 1
		} else {
			nodesLinkTypesCounters[targetNodeID][link.linkType]++
		}
	}

	for nodeID := range net.Nodes {
		node := net.Nodes[nodeID]
		if node.poiID > -1 {
			node.activityType = types.ACTIVITY_POI
			node.activityLinkType = types.LINK_UNDEFINED
		}
		if linkTypesCounters, ok := nodesLinkTypesCounters[nodeID]; ok {
			maxLinkTypes := []types.LinkType{}
			maxLinkTypeCount := 0
			for linkType, counter := range linkTypesCounters {
				if counter > maxLinkTypeCount {
					maxLinkTypeCount = counter
					maxLinkTypes = []types.LinkType{linkType}
				} else if counter == maxLinkTypeCount {
					maxLinkTypes = append(maxLinkTypes, linkType)
				}
			}
			if maxLinkTypeCount > 0 {
				node.activityType = types.ACTIVITY_LINK
				// When there are several link types pick the one with highest rank
				node.activityLinkType = types.FindPriorLinkType(maxLinkTypes)
			} else {
				node.activityType = types.ACTIVITY_NONE
				node.activityLinkType = types.LINK_UNDEFINED
			}
		}
	}

	for nodeID := range net.Nodes {
		node := net.Nodes[nodeID]
		node.boundaryType = types.BOUNDARY_NONE
		if node.activityType == types.ACTIVITY_POI {
			continue
		}
		if len(node.outcomingLinks) == 0 {
			node.boundaryType = types.BOUNDARY_INCOME_ONLY
		} else if len(node.incomingLinks) == 0 {
			node.boundaryType = types.BOUNDARY_OUTCOME_ONLY
		} else if len(node.incomingLinks) == 1 && (len(node.outcomingLinks) == 1) {
			incomingLink, ok := net.Links[node.incomingLinks[0]]
			if !ok {
				return fmt.Errorf("no incoming link with ID '%d'. Node ID: '%d'", node.incomingLinks[0], node.ID)
			}
			outcomingLink, ok := net.Links[node.outcomingLinks[0]]
			if !ok {
				return fmt.Errorf("no incoming link with ID '%d'. Node ID: '%d'", node.outcomingLinks[0], node.ID)
			}
			if incomingLink.sourceNodeID == outcomingLink.targetNodeID {
				node.boundaryType = types.BOUNDARY_INCOME_OUTCOME
			}
		}
	}
	for nodeID := range net.Nodes {
		node := net.Nodes[nodeID]
		if node.boundaryType == types.BOUNDARY_NONE {
			continue
		}
		node.zoneID = node.ID
	}
	return nil
}
