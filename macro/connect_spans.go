package macro

import (
	"sort"

	"github.com/LdDl/osm2gmns/geomath"
	"github.com/LdDl/osm2gmns/gmns"
)

func getSpansConnections(outcomingLink *Link, incomingLinksList []*Link) [][]connectionPair {
	// Sort outcoming links by angle in descending order (left to right)
	angles := make([]float64, len(incomingLinksList))
	for i, inLink := range incomingLinksList {
		angles[i] = geomath.AngleBetweenLines(inLink.geomEuclidean, outcomingLink.geomEuclidean)
	}
	indicesMap := make(map[gmns.LinkID]int, len(incomingLinksList))
	for index := range incomingLinksList {
		link := incomingLinksList[index]
		indicesMap[link.ID] = index
	}
	indices := make([]int, len(incomingLinksList))
	for i := range indices {
		indices[i] = i
	}
	sort.Slice(indices, func(i, j int) bool {
		return angles[indices[i]] > angles[indices[j]]
	})
	incomingLinksSorted := make([]*Link, len(incomingLinksList))
	for i := range incomingLinksSorted {
		incomingLinksSorted[i] = incomingLinksList[indices[i]]
	}
	// Evaluate lanes connections
	connections := make([][]connectionPair, len(incomingLinksSorted))
	outcomingLanes := outcomingLink.GetIncomingLanes()

	leftLink := incomingLinksSorted[0]
	leftLinkOutcomingLanes := leftLink.GetOutcomingLanes()

	minConnections := min(outcomingLanes, leftLinkOutcomingLanes)
	// In <-> Out
	connections[indicesMap[leftLink.ID]] = []connectionPair{
		{leftLinkOutcomingLanes - minConnections, leftLinkOutcomingLanes - 1},
		{0, minConnections - 1},
	}
	for i := range incomingLinksSorted[1:] {
		inLink := incomingLinksSorted[1:][i]
		lanesInfo := inLink.lanesInfo
		if len(lanesInfo.LanesList) == 0 {
			continue
		}
		inLinkOutcomingLanes := lanesInfo.LanesList[len(lanesInfo.LanesList)-1]
		minConnections := min(outcomingLanes, inLinkOutcomingLanes)
		// In <-> Out
		connections[indicesMap[inLink.ID]] = []connectionPair{{0, minConnections - 1}, {outcomingLanes - minConnections, outcomingLanes - 1}}
	}
	return connections
}
