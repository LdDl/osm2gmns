package osmmacro

import (
	"testing"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/go-gmns/gmns/types"
	"github.com/LdDl/osm2gmns/wrappers"
	"github.com/paulmach/orb"
	"github.com/paulmach/osm"
)

type geometryTestWay struct {
	id     osm.WayID
	nodes  []osm.NodeID
	oneWay bool
}

type geometryTestLink struct {
	nodes      []osm.NodeID
	geometryID gmns.GeometryID
	dirFlag    gmns.DirectionFlag
}

type geometryTestCase struct {
	name  string
	ways  []geometryTestWay
	links []geometryTestLink
}

func TestSegmentGeometryReferences(t *testing.T) {
	cases := []geometryTestCase{
		{
			name: "two-way road",
			ways: []geometryTestWay{{id: 10, nodes: []osm.NodeID{1, 2, 3}}},
			links: []geometryTestLink{
				{nodes: []osm.NodeID{1, 2, 3}, geometryID: gmns.GeometryIDFromInt64(0), dirFlag: gmns.DIR_FLAG_FORWARD},
				{nodes: []osm.NodeID{3, 2, 1}, geometryID: gmns.GeometryIDFromInt64(0), dirFlag: gmns.DIR_FLAG_REVERSE},
			},
		},
		{
			name: "one-way road",
			ways: []geometryTestWay{{id: 10, nodes: []osm.NodeID{1, 2, 3}, oneWay: true}},
			links: []geometryTestLink{
				{nodes: []osm.NodeID{1, 2, 3}, geometryID: gmns.GeometryIDFromInt64(0), dirFlag: gmns.DIR_FLAG_FORWARD},
			},
		},
		{
			name: "separate roundabout arcs within one way",
			ways: []geometryTestWay{{id: 1491559491, nodes: []osm.NodeID{1, 2, 3, 4, 1}, oneWay: true}},
			links: []geometryTestLink{
				{nodes: []osm.NodeID{1, 2, 3}, geometryID: gmns.GeometryIDFromInt64(0), dirFlag: gmns.DIR_FLAG_FORWARD},
				{nodes: []osm.NodeID{3, 4, 1}, geometryID: gmns.GeometryIDFromInt64(1), dirFlag: gmns.DIR_FLAG_FORWARD},
			},
		},
		{
			name: "parallel paths",
			ways: []geometryTestWay{
				{id: 10, nodes: []osm.NodeID{1, 2, 3}},
				{id: 11, nodes: []osm.NodeID{1, 4, 3}},
			},
			links: []geometryTestLink{
				{nodes: []osm.NodeID{1, 2, 3}, geometryID: gmns.GeometryIDFromInt64(0), dirFlag: gmns.DIR_FLAG_FORWARD},
				{nodes: []osm.NodeID{3, 2, 1}, geometryID: gmns.GeometryIDFromInt64(0), dirFlag: gmns.DIR_FLAG_REVERSE},
				{nodes: []osm.NodeID{1, 4, 3}, geometryID: gmns.GeometryIDFromInt64(1), dirFlag: gmns.DIR_FLAG_FORWARD},
				{nodes: []osm.NodeID{3, 4, 1}, geometryID: gmns.GeometryIDFromInt64(1), dirFlag: gmns.DIR_FLAG_REVERSE},
			},
		},
		{
			name: "repeated endpoints within one way",
			ways: []geometryTestWay{{id: 10, nodes: []osm.NodeID{1, 2, 3, 4, 1, 5, 3}}},
			links: []geometryTestLink{
				{nodes: []osm.NodeID{1, 2, 3}, geometryID: gmns.GeometryIDFromInt64(0), dirFlag: gmns.DIR_FLAG_FORWARD},
				{nodes: []osm.NodeID{3, 2, 1}, geometryID: gmns.GeometryIDFromInt64(0), dirFlag: gmns.DIR_FLAG_REVERSE},
				{nodes: []osm.NodeID{3, 4, 1}, geometryID: gmns.GeometryIDFromInt64(1), dirFlag: gmns.DIR_FLAG_FORWARD},
				{nodes: []osm.NodeID{1, 4, 3}, geometryID: gmns.GeometryIDFromInt64(1), dirFlag: gmns.DIR_FLAG_REVERSE},
				{nodes: []osm.NodeID{1, 5, 3}, geometryID: gmns.GeometryIDFromInt64(2), dirFlag: gmns.DIR_FLAG_FORWARD},
				{nodes: []osm.NodeID{3, 5, 1}, geometryID: gmns.GeometryIDFromInt64(2), dirFlag: gmns.DIR_FLAG_REVERSE},
			},
		},
		{
			name: "identical coordinates from independent ways",
			ways: []geometryTestWay{
				{id: 10, nodes: []osm.NodeID{1, 2, 3}},
				{id: 11, nodes: []osm.NodeID{1, 2, 3}},
			},
			links: []geometryTestLink{
				{nodes: []osm.NodeID{1, 2, 3}, geometryID: gmns.GeometryIDFromInt64(0), dirFlag: gmns.DIR_FLAG_FORWARD},
				{nodes: []osm.NodeID{3, 2, 1}, geometryID: gmns.GeometryIDFromInt64(0), dirFlag: gmns.DIR_FLAG_REVERSE},
				{nodes: []osm.NodeID{1, 2, 3}, geometryID: gmns.GeometryIDFromInt64(1), dirFlag: gmns.DIR_FLAG_FORWARD},
				{nodes: []osm.NodeID{3, 2, 1}, geometryID: gmns.GeometryIDFromInt64(1), dirFlag: gmns.DIR_FLAG_REVERSE},
			},
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			nodes := geometryTestNodes()
			ways := make([]*wrappers.WayOSM, len(testCase.ways))
			for index, source := range testCase.ways {
				ways[index] = &wrappers.WayOSM{
					ID:        source.id,
					Nodes:     source.nodes,
					IsOneWay:  source.oneWay,
					LinkType:  types.LINK_RESIDENTIAL,
					Capacity:  -1,
					FreeSpeed: -1,
					Tags:      wrappers.WayTags{Lanes: 3, LanesForward: 2, LanesBackward: 1, MaxSpeed: -1},
				}
			}
			network, err := NewNetFromOSM(ways, nodes)
			if err != nil {
				t.Fatal(err)
			}
			if len(network.Links) != len(testCase.links) {
				t.Fatalf("got %v links, expected %v", len(network.Links), len(testCase.links))
			}
			canonicalShapes := make(map[gmns.GeometryID]orb.LineString)
			for index, expected := range testCase.links {
				link := network.Links[gmns.LinkIDFromInt64(int64(index))]
				geometryID, hasGeometryID := link.GeometryID()
				dirFlag, hasDirFlag := link.DirFlag()
				if !hasGeometryID || geometryID != expected.geometryID {
					t.Fatalf("link %v: geometry ID %v, present %t, expected %v", link.ID, geometryID, hasGeometryID, expected.geometryID)
				}
				if !hasDirFlag || dirFlag != expected.dirFlag {
					t.Fatalf("link %v: direction %v, present %t, expected %v", link.ID, dirFlag, hasDirFlag, expected.dirFlag)
				}
				expectedShape := make(orb.LineString, len(expected.nodes))
				for pointIndex, nodeID := range expected.nodes {
					expectedShape[pointIndex] = nodes[nodeID].InnerNode.Point()
				}
				if !link.Geom().Equal(expectedShape) {
					t.Fatalf("link %v: travel geometry changed", link.ID)
				}
				if link.SourceOSMNode() != expected.nodes[0] || link.TargetOSMNode() != expected.nodes[len(expected.nodes)-1] {
					t.Fatalf("link %v: topology changed", link.ID)
				}
				if link.WasBidirectional() {
					expectedLanes := 2
					if dirFlag == gmns.DIR_FLAG_REVERSE {
						expectedLanes = 1
					}
					lanes, present := link.LanesNum()
					if !present || lanes != expectedLanes {
						t.Fatalf("link %v: direction-specific lanes changed", link.ID)
					}
				}
				canonical := link.Geom().Clone()
				if dirFlag == gmns.DIR_FLAG_REVERSE {
					canonical.Reverse()
				}
				previous, exists := canonicalShapes[geometryID]
				if exists && !canonical.Equal(previous) {
					t.Fatalf("geometry %v refers to different alignments", geometryID)
				}
				canonicalShapes[geometryID] = canonical
			}
		})
	}
}

func geometryTestNodes() map[osm.NodeID]*wrappers.NodeOSM {
	points := []orb.Point{{37, 55}, {37.001, 55.003}, {37.004, 55.004}, {37.006, 55.001}, {37.002, 55.006}}
	nodes := make(map[osm.NodeID]*wrappers.NodeOSM, len(points))
	for index, point := range points {
		nodeID := osm.NodeID(index + 1)
		nodes[nodeID] = &wrappers.NodeOSM{
			ID:         nodeID,
			InnerNode:  osm.Node{ID: nodeID, Lon: point[0], Lat: point[1]},
			IsCrossing: nodeID == 1 || nodeID == 3,
		}
	}
	return nodes
}
