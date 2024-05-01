package macro

import (
	"fmt"

	"github.com/LdDl/osm2gmns/wrappers"
	"github.com/paulmach/osm"
	"github.com/pkg/errors"
)

func NewNetFromOSM(ways []*wrappers.WayOSM, nodesSet map[osm.NodeID]*wrappers.NodeOSM) error {
	for i := range ways {
		way := ways[i]
		if way.IsPureCycle {
			continue
		}
		segments, err := prepareSegments(way, nodesSet)
		if err != nil {
			return errors.Wrapf(err, "Can't prepare segments for way: %d", way.ID)
		}
		for j := range segments {
			segment := segments[j]
			if len(segment) < 2 {
				continue
			}
			// @todo: Prepare nodes and links
		}
	}
	return nil
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
				return segments, fmt.Errorf("No such node: %d", nextNodeID)
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
