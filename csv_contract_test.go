package osm2gmns_test

import (
	"encoding/csv"
	"encoding/json"
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
	"github.com/paulmach/orb"
	"github.com/stretchr/testify/assert"
)

func readCSVRecords(t *testing.T, directory, filename string) []map[string]string {
	t.Helper()
	file, err := os.Open(filepath.Join(directory, filename))
	assert.NoError(t, err)
	if err != nil {
		t.FailNow()
	}
	defer file.Close()
	records, err := csv.NewReader(file).ReadAll()
	assert.NoError(t, err)
	if err != nil || len(records) == 0 {
		t.FailNow()
	}
	rows := make([]map[string]string, 0, len(records)-1)
	for _, record := range records[1:] {
		row := make(map[string]string)
		for index, name := range records[0] {
			row[name] = record[index]
		}
		rows = append(rows, row)
	}
	return rows
}

func authoredNetwork(t *testing.T) *macro.Net {
	t.Helper()
	first := gmns.NodeIDFromString("001")
	last := gmns.NodeIDFromString("1")
	road := gmns.LinkIDFromString("road")
	back := gmns.LinkIDFromString("return")
	network := macro.NewNet()
	config := gmns.NewConfig(gmns.IDTypeString, gmns.WithCurrency("EUR"))
	assert.NoError(t, network.SetConfig(config))
	network.Nodes[first] = macro.NewNodeFrom(first, macro.WithNodeName("Start, \"quoted\""))
	network.Nodes[last] = macro.NewNodeFrom(last, macro.WithZoneID(first))
	assert.NoError(t, network.AddZone(gmns.NewZone(first)))
	assert.NoError(t, network.AddUseDefinition(gmns.NewUseDefinition("custom", 2, 1.25)))
	groupUses := []string{"custom"}
	assert.NoError(t, network.AddUseGroup(gmns.NewUseGroup("group", groupUses)))
	network.Links[road] = macro.NewLinkFrom(road, first, last,
		macro.WithCapacity(12.75),
		macro.WithFreeSpeed(0),
		macro.WithLanesNum(0),
		macro.WithToll(5),
		macro.WithAllowedUses("group"),
		macro.WithGrade(-1.5),
		macro.WithRightOfWayWidthMeters(0),
	)
	network.Links[back] = macro.NewLinkFrom(back, last, first)
	maneuver := movement.NewMovement(gmns.MovementIDFromString("001"), last, road, back, "NBL", "left",
		movement.WithPenalty(0),
		movement.WithCapacity(12.75),
		movement.WithIncomeLaneStart(-1),
		movement.WithAllowedUses("group"),
		movement.WithControlType("custom control"),
	)
	assert.NoError(t, network.AddMovement(maneuver))
	laneID := gmns.LaneIDFromString("lane-001")
	lane := gmns.NewLane(laneID, road, -1, gmns.WithLaneWidthMeters(0), gmns.WithLaneAllowedUses("custom"))
	assert.NoError(t, network.AddLane(lane))
	segmentID := gmns.SegmentIDFromString("segment-001")
	segment := gmns.NewSegment(segmentID, road, first, 0, 1,
		gmns.WithSegmentGrade(-2),
		gmns.WithSegmentFreeSpeed(0),
		gmns.WithSegmentToll(0),
	)
	assert.NoError(t, network.AddSegment(segment))
	segmentLane := gmns.NewSegmentLane(gmns.SegmentLaneIDFromString("detail"), segmentID, -1,
		gmns.WithSegmentLaneParentLane(laneID),
		gmns.WithSegmentLaneWidthMeters(2.5),
	)
	assert.NoError(t, network.AddSegmentLane(segmentLane))
	return network
}

func TestStandardDatasetPreservesAuthoredRecords(t *testing.T) {
	network := authoredNetwork(t)
	before, err := osmmacro.BuildDataset(network, osmmacro.GEOMETRY_INLINE)
	assert.NoError(t, err)
	directory := t.TempDir()
	assert.NoError(t, osmmacro.ExportToCSV(network, directory))
	nodes := readCSVRecords(t, directory, "node_macro.csv")
	assert.Equal(t, "001", nodes[0]["node_id"])
	assert.Equal(t, "1", nodes[1]["node_id"])
	assert.Equal(t, "Start, \"quoted\"", nodes[0]["name"])
	assert.Equal(t, "001", nodes[1]["zone_id"])
	links := readCSVRecords(t, directory, "link_macro.csv")
	road := links[1]
	assert.Equal(t, "road", road["link_id"])
	assert.Equal(t, "001", road["from_node_id"])
	assert.Equal(t, "1", road["to_node_id"])
	assert.Equal(t, "12.75", road["capacity"])
	assert.Equal(t, "0", road["free_speed"])
	assert.Equal(t, "0", road["lanes"])
	assert.Equal(t, "-1.5", road["grade"])
	assert.Equal(t, "5", road["toll"])
	assert.Equal(t, "group", road["allowed_uses"])
	assert.Empty(t, road["length"])
	assert.NotContains(t, road, "source_node")
	assert.NotContains(t, road, "macro_link_id")
	movements := readCSVRecords(t, directory, "movement_macro.csv")
	assert.Equal(t, "001", movements[0]["mvmt_id"])
	assert.Equal(t, "0", movements[0]["penalty"])
	assert.Equal(t, "12.75", movements[0]["capacity"])
	assert.Equal(t, "-1", movements[0]["start_ib_lane"])
	assert.Empty(t, movements[0]["end_ib_lane"])
	assert.Empty(t, movements[0]["start_ob_lane"])
	assert.Equal(t, "custom control", movements[0]["ctrl_type"])
	assert.Equal(t, "group", movements[0]["allowed_uses"])
	assert.Equal(t, "NBL", movements[0]["mvmt_code"])
	assert.Equal(t, "-1", readCSVRecords(t, directory, "lane_macro.csv")[0]["lane_num"])
	assert.Equal(t, "0", readCSVRecords(t, directory, "lane_macro.csv")[0]["width"])
	assert.Equal(t, "-2", readCSVRecords(t, directory, "segment_macro.csv")[0]["grade"])
	assert.Equal(t, "lane-001", readCSVRecords(t, directory, "segment_lane_macro.csv")[0]["parent_lane_id"])
	config := readCSVRecords(t, directory, "config_macro.csv")[0]
	assert.Equal(t, "string", config["id_type"])
	assert.Equal(t, "EUR", config["currency"])
	assert.Equal(t, "meter", config["long_length"])
	assert.Equal(t, "km/h", config["speed"])
	assert.Equal(t, "EPSG:4326", config["crs"])
	after, err := osmmacro.BuildDataset(network, osmmacro.GEOMETRY_INLINE)
	assert.NoError(t, err)
	assert.Equal(t, before, after, "export must not modify records")
	firstFiles := snapshotFiles(t, directory)
	assert.NoError(t, osmmacro.ExportToCSV(network, directory))
	assert.Equal(t, firstFiles, snapshotFiles(t, directory), "export must be deterministic")
	data := firstFiles["datapackage_macro.json"]
	descriptor := make(map[string]any)
	assert.NoError(t, json.Unmarshal(data, &descriptor))
	assert.Len(t, descriptor["resources"], 10)
}

func snapshotFiles(t *testing.T, directory string) map[string][]byte {
	t.Helper()
	entries, err := os.ReadDir(directory)
	assert.NoError(t, err)
	result := make(map[string][]byte)
	for _, entry := range entries {
		assert.False(t, entry.IsDir())
		data, readErr := os.ReadFile(filepath.Join(directory, entry.Name()))
		assert.NoError(t, readErr)
		result[entry.Name()] = data
	}
	return result
}

func TestLevelsHaveIndependentResources(t *testing.T) {
	directory := t.TempDir()
	first := gmns.NodeIDFromString("001")
	last := gmns.NodeIDFromString("1")
	road := gmns.LinkIDFromString("road")
	config := gmns.NewConfig(gmns.IDTypeString)
	mesoNet := meso.NewNet()
	assert.NoError(t, mesoNet.SetConfig(config))
	mesoNet.Nodes[first] = meso.NewNodeFrom(first)
	mesoNet.Nodes[last] = meso.NewNodeFrom(last)
	mesoNet.Links[road] = meso.NewLinkFrom(road, first, last, meso.WithCapacity(12.75))
	microNet := micro.NewNet()
	assert.NoError(t, microNet.SetConfig(config))
	assert.NoError(t, microNet.AddNode(micro.NewNodeFrom(first)))
	assert.NoError(t, microNet.AddNode(micro.NewNodeFrom(last)))
	assert.NoError(t, microNet.AddLink(micro.NewLinkFrom(road, first, last, micro.WithFreeSpeed(0))))
	assert.NoError(t, osmmeso.ExportToCSV(mesoNet, directory))
	assert.NoError(t, osmmicro.ExportToCSV(microNet, directory))
	assert.Equal(t, "12.75", readCSVRecords(t, directory, "link_meso.csv")[0]["capacity"])
	assert.Empty(t, readCSVRecords(t, directory, "link_micro.csv")[0]["capacity"])
	assert.Equal(t, "0", readCSVRecords(t, directory, "link_micro.csv")[0]["free_speed"])
	assert.FileExists(t, filepath.Join(directory, "datapackage_meso.json"))
	assert.FileExists(t, filepath.Join(directory, "datapackage_micro.json"))
	assert.NoFileExists(t, filepath.Join(directory, "movement_meso.csv"))
	assert.NoFileExists(t, filepath.Join(directory, "movement_micro.csv"))
}

func TestInvalidDatasetDoesNotReplaceOutput(t *testing.T) {
	network := authoredNetwork(t)
	directory := t.TempDir()
	assert.NoError(t, osmmacro.ExportToCSV(network, directory))
	original := snapshotFiles(t, directory)
	delete(network.Nodes, gmns.NodeIDFromString("001"))
	assert.Error(t, osmmacro.ExportToCSV(network, directory))
	assert.Equal(t, original, snapshotFiles(t, directory))
}

func TestUndirectedInlineGeometry(t *testing.T) {
	network := authoredNetwork(t)
	link := network.Links[gmns.LinkIDFromString("road")]
	shape := orb.LineString{{0, 0}, {1, 1}}
	macro.WithLineGeom(shape)(link)
	macro.WithDirected(false)(link)
	directory := t.TempDir()
	assert.NoError(t, osmmacro.ExportToCSV(network, directory))
	row := readCSVRecords(t, directory, "link_macro.csv")[1]
	assert.Equal(t, "false", row["directed"])
	assert.Equal(t, "0", row["dir_flag"])
}

func TestLargeIntegerIdentifiersStayExact(t *testing.T) {
	net := macro.NewNet()
	assert.NoError(t, net.SetConfig(gmns.NewConfig(gmns.IDTypeInteger)))
	largeID, err := gmns.ParseNodeID("922337203685477580812345", gmns.IDTypeInteger)
	assert.NoError(t, err)
	negativeID := gmns.NodeIDFromInt64(-1)
	net.Nodes[negativeID] = macro.NewNodeFrom(negativeID)
	net.Nodes[largeID] = macro.NewNodeFrom(largeID)
	road := gmns.LinkIDFromInt64(0)
	net.Links[road] = macro.NewLinkFrom(road, negativeID, largeID)
	directory := t.TempDir()
	assert.NoError(t, osmmacro.ExportToCSV(net, directory))
	nodes := readCSVRecords(t, directory, "node_macro.csv")
	assert.Equal(t, "-1", nodes[0]["node_id"])
	assert.Equal(t, "922337203685477580812345", nodes[1]["node_id"])
	row := readCSVRecords(t, directory, "link_macro.csv")[0]
	assert.Equal(t, "0", row["link_id"])
	assert.Equal(t, "922337203685477580812345", row["to_node_id"])
}

func TestUnsupportedAuthoredMovementCodeIsNotDiscarded(t *testing.T) {
	net := authoredNetwork(t)
	identifier := gmns.MovementIDFromString("001")
	record, found := net.Movement(identifier)
	assert.True(t, found)
	movement.WithMvmtTextID("CUSTOM")(record)
	assert.NoError(t, net.UpdateMovement(record))
	directory := t.TempDir()
	assert.Error(t, osmmacro.ExportToCSV(net, directory))
	stored, found := net.Movement(identifier)
	assert.True(t, found)
	assert.Equal(t, movement.MovementCompositeType("CUSTOM"), stored.MvmtTextID())
	entries, err := os.ReadDir(directory)
	assert.NoError(t, err)
	assert.Empty(t, entries)
}
