package osmmacro

import (
	"fmt"
	"sort"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/go-gmns/gmns/types"
	"github.com/LdDl/go-gmns/macro"
	"github.com/LdDl/osm2gmns/wrappers"
	"github.com/paulmach/osm"
	"github.com/pkg/errors"
)

func NewNetFromOSM(ways []*wrappers.WayOSM, nodesSet map[osm.NodeID]*wrappers.NodeOSM) (*macro.Net, error) {
	return newNetFromOSM(ways, nodesSet, nil)
}

// NewNetFromOSMWithEvidence captures source spans while constructing the network.
func NewNetFromOSMWithEvidence(ways []*wrappers.WayOSM, nodesSet map[osm.NodeID]*wrappers.NodeOSM) (*macro.Net, *SourceEvidence, error) {
	evidence := newSourceEvidence()
	net, err := newNetFromOSM(ways, nodesSet, evidence)
	if err != nil {
		return nil, nil, err
	}
	return net, evidence, nil
}

func newNetFromOSM(ways []*wrappers.WayOSM, nodesSet map[osm.NodeID]*wrappers.NodeOSM, evidence *SourceEvidence) (*macro.Net, error) {

	elevations, err := readNodeElevations(ways, nodesSet)
	if err != nil {
		return nil, err
	}

	linkIDs := gmns.IDAllocator{}
	lastLinkID := linkIDs.NextLinkID()
	nodeIDs := gmns.IDAllocator{}
	lastNodeID := nodeIDs.NextNodeID()
	geometryIDs := gmns.IDAllocator{}
	lastGeometryID := geometryIDs.NextGeometryID()

	observed := make(map[osm.NodeID]gmns.NodeID)
	nodes := make(map[gmns.NodeID]*macro.Node)
	links := make(map[gmns.LinkID]*macro.Link)
	geometries := make([]*gmns.Geometry, 0)

	for i := range ways {
		way := ways[i]
		if way.IsPureCycle {
			continue
		}
		segments, err := prepareSegments(way, nodesSet)
		if err != nil {
			return nil, errors.Wrapf(err, "can't prepare segments for way: %v", way.ID)
		}
		startIndex := 0
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
					return nil, fmt.Errorf("no such source node '%v'. Way ID: '%v'", sourceNodeID, way.ID)
				}
				nodes[lastNodeID] = newNodeFromElevation(lastNodeID, sourceNode, elevations[sourceNodeID].meters)
				observed[sourceNodeID] = lastNodeID
				currentSourceNodeID = lastNodeID
				lastNodeID = nodeIDs.NextNodeID()
			} else {
				currentSourceNodeID = nID
			}
			targetNodeID := segment[len(segment)-1]
			if nID, ok := observed[targetNodeID]; !ok {
				targetNode, ok := nodesSet[targetNodeID]
				if !ok {
					return nil, fmt.Errorf("no such target node '%v'. Way ID: '%v'", targetNodeID, way.ID)
				}
				nodes[lastNodeID] = newNodeFromElevation(lastNodeID, targetNode, elevations[targetNodeID].meters)
				observed[targetNodeID] = lastNodeID
				currentTargetNodeID = lastNodeID
				lastNodeID = nodeIDs.NextNodeID()
			} else {
				currentTargetNodeID = nID
			}

			/* Create links */
			nodesForSegment := make([]*wrappers.NodeOSM, len(segment))
			for i, nodeID := range segment {
				nodesForSegment[i] = nodesSet[nodeID]
			}
			link, err := NewLinkFrom(lastLinkID, currentSourceNodeID, currentTargetNodeID, nodes[currentSourceNodeID].OSMNode(), nodes[currentTargetNodeID].OSMNode(), macro.DIRECTION_FORWARD, way, nodesForSegment)
			if err != nil {
				return nil, err
			}
			links[lastLinkID] = link
			geometries = append(geometries, gmns.NewGeometry(lastGeometryID, link.Geom()))
			// Reference the source segment's forward alignment; Geom() stays in travel order.
			macro.WithGeometryID(lastGeometryID)(links[lastLinkID])
			macro.WithDirFlag(gmns.DIR_FLAG_FORWARD)(links[lastLinkID])
			macro.WithOutcomingLinks(lastLinkID)(nodes[currentSourceNodeID])
			macro.WithIncomingLinks(lastLinkID)(nodes[currentTargetNodeID])
			lastLinkID = linkIDs.NextLinkID()
			var backwardLink *macro.Link
			if !way.IsOneWay {
				link, err := NewLinkFrom(lastLinkID, currentTargetNodeID, currentSourceNodeID, nodes[currentTargetNodeID].OSMNode(), nodes[currentSourceNodeID].OSMNode(), macro.DIRECTION_BACKWARD, way, nodesForSegment)
				if err != nil {
					return nil, err
				}
				links[lastLinkID] = link
				backwardLink = link
				macro.WithGeometryID(lastGeometryID)(links[lastLinkID])
				macro.WithDirFlag(gmns.DIR_FLAG_REVERSE)(links[lastLinkID])
				macro.WithOutcomingLinks(lastLinkID)(nodes[currentTargetNodeID])
				macro.WithIncomingLinks(lastLinkID)(nodes[currentSourceNodeID])
				lastLinkID = linkIDs.NextLinkID()
			}
			if evidence != nil {
				evidence.captureWay(way)
				evidence.captureLink(link, way, startIndex, startIndex+len(segment)-1, 1)
				if !way.IsOneWay {
					evidence.captureLink(backwardLink, way, startIndex, startIndex+len(segment)-1, -1)
				}
			}
			startIndex += len(segment) - 1
			lastGeometryID = geometryIDs.NextGeometryID()
		}
	}

	net := &macro.Net{Nodes: nodes, Links: links}
	for _, geometry := range geometries {
		err := net.AddGeometry(geometry)
		if err != nil {
			return nil, err
		}
	}
	configErr := net.SetConfig(gmns.NewConfig(gmns.IDTypeInteger))
	if configErr != nil {
		return nil, configErr
	}
	err = genBoundaryAndActivityType(net)
	if err != nil {
		return nil, err
	}
	if evidence != nil {
		evidence.captureNodes(net, nodesSet, elevations)
	} else {
		warnNodeElevations(elevations)
	}
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
				return segments, fmt.Errorf("no such node: %v", nextNodeID)
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
			return fmt.Errorf("no source node with ID '%v'. Link ID: '%v'", sourceNodeID, link.ID)
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
			return fmt.Errorf("no target node with ID '%v'. Link ID: '%v'", targetNodeID, link.ID)
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
			// Sort link types for deterministic iteration
			sortedLinkTypes := make([]types.LinkType, 0, len(linkTypesCounters))
			for linkType := range linkTypesCounters {
				sortedLinkTypes = append(sortedLinkTypes, linkType)
			}
			sort.Slice(sortedLinkTypes, func(i, j int) bool {
				return sortedLinkTypes[i] < sortedLinkTypes[j]
			})
			for _, linkType := range sortedLinkTypes {
				counter := linkTypesCounters[linkType]
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
				return fmt.Errorf("no incoming link with ID '%v'. Node ID: '%v'", incomeLinks[0], node.ID)
			}
			outcomingLink, ok := macroNet.Links[outcomeLinks[0]]
			if !ok {
				return fmt.Errorf("no incoming link with ID '%v'. Node ID: '%v'", outcomeLinks[0], node.ID)
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
		err := macroNet.AddZone(gmns.NewZone(node.ID))
		if err != nil {
			return err
		}
	}
	return nil
}
