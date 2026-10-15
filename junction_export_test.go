package osm2gmns_test

import (
	"testing"

	"github.com/LdDl/go-gmns/generators"
	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/go-gmns/gmns/types"
	"github.com/LdDl/go-gmns/macro"
	"github.com/LdDl/go-gmns/utils/geomath"
	"github.com/LdDl/osm2gmns/csvexport"
	"github.com/LdDl/osm2gmns/osmmacro"
	"github.com/LdDl/osm2gmns/osmmeso"
	"github.com/LdDl/osm2gmns/osmmicro"
	"github.com/LdDl/osm2gmns/preparation"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geo"
	"github.com/stretchr/testify/assert"
)

type junctionExportCase struct {
	name       string
	control    types.ControlType
	connectors int
}

func TestJunctionExportOwnership(t *testing.T) {
	cases := []junctionExportCase{
		{
			name:       "uncontrolled",
			control:    types.CONTROL_TYPE_NO_CONTROL,
			connectors: 0,
		},
		{
			name:       "signal",
			control:    types.CONTROL_TYPE_IS_SIGNAL,
			connectors: 1,
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			network := junctionNetwork(t, testCase.control)
			movements, err := generators.GenerateMovements(network)
			assert.NoError(t, err)
			if err != nil {
				return
			}
			mesoNet, err := generators.GenerateMesoscopic(network, movements)
			assert.NoError(t, err)
			if err != nil {
				return
			}
			microNet, err := generators.GenerateMicroscopic(network, mesoNet, movements)
			assert.NoError(t, err)
			if err != nil {
				return
			}
			connectors := 0
			for _, link := range mesoNet.Links {
				if link.MacroNode() == gmns.NodeIDFromInt64(2) {
					connectors++
				}
			}
			assert.Equal(t, testCase.connectors, connectors)
			_, err = preparation.StoreGeneratedMovements(network, movements)
			assert.NoError(t, err)
			options := preparation.DefaultUseOptions()
			assert.NoError(t, preparation.Macro(network, options))
			assert.NoError(t, preparation.Meso(mesoNet, options))
			assert.NoError(t, preparation.Micro(microNet, options))
			macroDataset, err := osmmacro.BuildDataset(network, osmmacro.GEOMETRY_INLINE)
			assert.NoError(t, err)
			mesoDataset, err := osmmeso.BuildDataset(mesoNet)
			assert.NoError(t, err)
			microDataset, err := osmmicro.BuildDataset(microNet)
			assert.NoError(t, err)
			datasets := []csvexport.Dataset{
				macroDataset,
				mesoDataset,
				microDataset,
			}
			directory := t.TempDir()
			assert.NoError(t, csvexport.WriteDatasets(directory, datasets))
			rows := readCSVRecords(t, directory, "movement_macro.csv")
			assert.Len(t, rows, 1)
			assert.Equal(t, "2", rows[0]["node_id"])
			assert.Equal(t, "10", rows[0]["ib_link_id"])
			assert.Equal(t, "11", rows[0]["ob_link_id"])
			assert.Equal(t, testCase.control.String(), rows[0]["ctrl_type"])
			assert.Empty(t, mesoNet.MovementRecords())
			assert.Empty(t, microNet.MovementRecords())
			for _, level := range []string{"meso", "micro"} {
				links := readCSVRecords(t, directory, "link_"+level+".csv")
				nodes := readCSVRecords(t, directory, "node_"+level+".csv")
				nodeIDs := make(map[string]bool)
				for _, node := range nodes {
					nodeIDs[node["node_id"]] = true
				}
				for _, link := range links {
					assert.True(t, nodeIDs[link["from_node_id"]])
					assert.True(t, nodeIDs[link["to_node_id"]])
					assert.Empty(t, link["parent_link_id"])
					assert.NotContains(t, link, "movement_id")
					assert.NotContains(t, link, "macro_link_id")
				}
			}
		})
	}
}

func junctionNetwork(t *testing.T, controlType types.ControlType) *macro.Net {
	t.Helper()
	points := []orb.Point{
		{30, 60},
		{30.002, 60},
		{30.004, 60},
	}
	network := macro.NewNet()
	assert.NoError(t, network.SetConfig(gmns.NewConfig(gmns.IDTypeInteger)))
	network.Nodes[gmns.NodeIDFromInt64(1)] = macro.NewNodeFrom(gmns.NodeIDFromInt64(1),
		macro.WithPointGeom(points[0]),
		macro.WithOutcomingLinks(gmns.LinkIDFromInt64(10)),
	)
	network.Nodes[gmns.NodeIDFromInt64(2)] = macro.NewNodeFrom(gmns.NodeIDFromInt64(2),
		macro.WithPointGeom(points[1]),
		macro.WithIncomingLinks(gmns.LinkIDFromInt64(10)),
		macro.WithOutcomingLinks(gmns.LinkIDFromInt64(11)),
		macro.WithNodeControlType(controlType),
	)
	network.Nodes[gmns.NodeIDFromInt64(3)] = macro.NewNodeFrom(gmns.NodeIDFromInt64(3),
		macro.WithPointGeom(points[2]),
		macro.WithIncomingLinks(gmns.LinkIDFromInt64(11)),
	)
	links := []*macro.Link{
		macro.NewLinkFrom(gmns.LinkIDFromInt64(10), gmns.NodeIDFromInt64(1), gmns.NodeIDFromInt64(2), macro.WithLineGeom(orb.LineString{points[0], points[1]})),
		macro.NewLinkFrom(gmns.LinkIDFromInt64(11), gmns.NodeIDFromInt64(2), gmns.NodeIDFromInt64(3), macro.WithLineGeom(orb.LineString{points[1], points[2]})),
	}
	for _, link := range links {
		macro.WithLineGeomEuclidean(geomath.LineToEuclidean(link.Geom()))(link)
		macro.WithLengthMeters(geo.LengthHaversine(link.Geom()))(link)
		macro.WithLanesNum(2)(link)
		laneInfo, err := macro.NewLanesInfo(link)
		assert.NoError(t, err)
		if err != nil {
			t.FailNow()
		}
		macro.WithLanesInfo(laneInfo)(link)
		macro.WithFreeSpeed(40)(link)
		macro.WithCapacity(1800)(link)
		macro.WithAllowedAgentTypes([]types.AgentType{types.AGENT_AUTO})(link)
		network.Links[link.ID] = link
	}
	return network
}
