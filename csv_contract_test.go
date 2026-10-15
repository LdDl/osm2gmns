package osm2gmns_test

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/go-gmns/macro"
	"github.com/LdDl/go-gmns/meso"
	"github.com/LdDl/go-gmns/micro"
	"github.com/LdDl/go-gmns/movement"
	"github.com/LdDl/osm2gmns/osmmacro"
	"github.com/LdDl/osm2gmns/osmmeso"
	"github.com/LdDl/osm2gmns/osmmicro"
	"github.com/LdDl/osm2gmns/osmmovement"
	"github.com/stretchr/testify/assert"
)

type csvHeaderCase struct {
	filename string
	columns  []string
}

func readCSVRecords(t *testing.T, directory, filename string) [][]string {
	t.Helper()
	file, err := os.Open(filepath.Join(directory, filename))
	assert.NoError(t, err)
	if err != nil {
		t.FailNow()
	}
	defer file.Close()
	reader := csv.NewReader(file)
	reader.Comma = ';'
	records, err := reader.ReadAll()
	assert.NoError(t, err)
	if err != nil {
		t.FailNow()
	}
	return records
}

func TestExistingCSVColumnsAndExactValues(t *testing.T) {
	directory := t.TempDir()
	first := gmns.NodeIDFromString("001")
	last := gmns.NodeIDFromString("1")
	road := gmns.LinkIDFromString("road")
	macroNet := macro.NewNet()
	macroNet.Nodes[first] = macro.NewNodeFrom(first)
	macroNet.Nodes[last] = macro.NewNodeFrom(last, macro.WithZoneID(gmns.NodeIDFromString("001")))
	macroNet.Links[road] = macro.NewLinkFrom(road, first, last, macro.WithCapacity(12.75), macro.WithFreeSpeed(0), macro.WithLanesNum(0), macro.WithToll(5), macro.WithAllowedUses("custom"))
	assert.NoError(t, osmmacro.ExportToCSV(macroNet, directory))
	mesoNet := meso.NewNet()
	mesoNet.Nodes[first] = meso.NewNodeFrom(first)
	mesoNet.Nodes[last] = meso.NewNodeFrom(last)
	mesoNet.Links[road] = meso.NewLinkFrom(road, first, last, meso.WithCapacity(12.75), meso.WithFreeSpeed(0))
	assert.NoError(t, osmmeso.ExportToCSV(mesoNet, directory))
	microNet := micro.NewNet()
	assert.NoError(t, microNet.AddNode(micro.NewNodeFrom(first)))
	assert.NoError(t, microNet.AddNode(micro.NewNodeFrom(last)))
	assert.NoError(t, microNet.AddLink(micro.NewLinkFrom(road, first, last, micro.WithCapacity(12.75), micro.WithFreeSpeed(0))))
	assert.NoError(t, osmmicro.ExportToCSV(microNet, directory))
	identifier := gmns.MovementIDFromString("001")
	mvmt := movement.NewMovement(identifier, last, road, road, "", "", movement.WithPenalty(0), movement.WithCapacity(12.75), movement.WithIncomeLaneStart(-1))
	storage := movement.MovementsStorage{identifier: mvmt}
	assert.NoError(t, osmmovement.ExportToCSV(storage, filepath.Join(directory, "movement.csv")))
	cases := []csvHeaderCase{
		{
			filename: "node.csv",
			columns:  []string{"id", "osm_node_id", "control_type", "boundary_type", "activity_type", "activity_link_type", "zone_id", "intersection_id", "poi_id", "osm_highway", "name", "longitude", "latitude"},
		},
		{
			filename: "link.csv",
			columns:  []string{"id", "source_node", "target_node", "osm_way_id", "source_osm_node_id", "target_osm_node_id", "link_class", "is_link", "link_type", "control_type", "allowed_agent_types", "was_bidirectional", "lanes", "max_speed", "free_speed", "capacity", "length_meters", "name", "geom"},
		},
		{
			filename: "mesonode.csv",
			columns:  []string{"id", "zone_id", "macro_node_id", "macro_link_id", "activity_link_type", "boundary_type", "longitude", "latitude"},
		},
		{
			filename: "mesolink.csv",
			columns:  []string{"id", "source_node", "target_node", "macro_node_id", "macro_link_id", "link_type", "control_type", "movement_id", "movement_composite_type", "allowed_agent_types", "lanes", "free_speed", "capacity", "length_meters", "geom"},
		},
		{
			filename: "micronode.csv",
			columns:  []string{"id", "meso_link_id", "lane_id", "cell_index", "is_upstream_end", "is_downstream_end", "zone_id", "boundary_type", "longitude", "latitude"},
		},
		{
			filename: "microlink.csv",
			columns:  []string{"id", "source_node", "target_node", "meso_link_id", "macro_link_id", "macro_node_id", "cell_type", "lane_id", "is_first_movement_cell", "movement_composite_type", "additional_travel_cost", "meso_link_type", "control_type", "allowed_agent_types", "free_speed", "capacity", "length_meters", "geom"},
		},
		{
			filename: "movement.csv",
			columns:  []string{"id", "node_id", "osm_node_id", "name", "in_link_id", "in_lane_start", "in_lane_end", "out_link_id", "out_lane_start", "out_lane_end", "lanes_num", "from_osm_node_id", "to_osm_node_id", "type", "penalty", "capacity", "control_type", "movement_composite_type", "volume", "free_speed", "allowed_agent_types", "geom"},
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.filename, func(t *testing.T) {
			records := readCSVRecords(t, directory, testCase.filename)
			assert.Equal(t, testCase.columns, records[0], "no additional CSV columns in this migration")
			for _, row := range records[1:] {
				assert.Len(t, row, len(testCase.columns))
			}
		})
	}
	entries, err := os.ReadDir(directory)
	assert.NoError(t, err)
	assert.Len(t, entries, 7, "no new resource files in the default export")
	for _, filename := range []string{"link.csv", "mesolink.csv", "microlink.csv"} {
		records := readCSVRecords(t, directory, filename)
		row := make(map[string]string)
		for index, name := range records[0] {
			row[name] = records[1][index]
		}
		assert.Equal(t, "001", row["source_node"])
		assert.Equal(t, "1", row["target_node"])
		assert.Equal(t, "12.75", row["capacity"])
		assert.Equal(t, "0", row["free_speed"])
		assert.Equal(t, "", row["length_meters"])
	}
	records := readCSVRecords(t, directory, "movement.csv")
	row := make(map[string]string)
	for index, name := range records[0] {
		row[name] = records[1][index]
	}
	assert.Equal(t, "001", row["id"])
	assert.Equal(t, "0", row["penalty"])
	assert.Equal(t, "12.75", row["capacity"])
	assert.Equal(t, "-1", row["in_lane_start"])
	assert.Empty(t, row["in_lane_end"])
	assert.Empty(t, row["out_lane_start"])
	assert.Empty(t, row["volume"])
	movement.WithoutPenalty()(mvmt)
	movement.WithoutCapacity()(mvmt)
	assert.NoError(t, osmmovement.ExportToCSV(storage, filepath.Join(directory, "movement.csv")))
	records = readCSVRecords(t, directory, "movement.csv")
	for index, name := range records[0] {
		row[name] = records[1][index]
	}
	assert.Empty(t, row["penalty"])
	assert.Empty(t, row["capacity"])
}
