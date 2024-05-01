package macro

import (
	"fmt"

	"github.com/LdDl/osm2gmns/wrappers"
	"github.com/paulmach/osm"
	"github.com/pkg/errors"
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
					return nil, fmt.Errorf("No such source node '%d'. Way ID: '%d'", sourceNodeID, way.ID)
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
					return nil, fmt.Errorf("No such target node '%d'. Way ID: '%d'", targetNodeID, way.ID)
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

	return &Net{Nodes: nodes, Links: links}, nil
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
