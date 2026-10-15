package osmmacro

import (
	"bytes"
	"encoding/csv"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/go-gmns/macro"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/encoding/wkt"
	"github.com/stretchr/testify/assert"
)

func geometryExportNetwork(t *testing.T) *macro.Net {
	t.Helper()
	network := macro.NewNet()
	forward := orb.LineString{{37, 55}, {37.003, 55.002}, {37.01, 55.01}}
	backward := forward.Clone()
	backward.Reverse()
	network.Nodes[gmns.NodeIDFromInt64(1)] = macro.NewNodeFrom(gmns.NodeIDFromInt64(1), macro.WithPointGeom(forward[0]))
	network.Nodes[gmns.NodeIDFromInt64(2)] = macro.NewNodeFrom(gmns.NodeIDFromInt64(2), macro.WithPointGeom(forward[len(forward)-1]))
	network.Links[gmns.LinkIDFromInt64(20)] = macro.NewLinkFrom(gmns.LinkIDFromInt64(20), gmns.NodeIDFromInt64(1), gmns.NodeIDFromInt64(2), macro.WithLineGeom(forward), macro.WithGeometryID(gmns.GeometryIDFromInt64(7)), macro.WithDirFlag(gmns.DIR_FLAG_FORWARD), macro.WithLanesNum(2), macro.WithFreeSpeed(40))
	network.Links[gmns.LinkIDFromInt64(21)] = macro.NewLinkFrom(gmns.LinkIDFromInt64(21), gmns.NodeIDFromInt64(2), gmns.NodeIDFromInt64(1), macro.WithLineGeom(backward), macro.WithGeometryID(gmns.GeometryIDFromInt64(7)), macro.WithDirFlag(gmns.DIR_FLAG_REVERSE), macro.WithLanesNum(1), macro.WithFreeSpeed(30))
	err := network.AddGeometry(gmns.NewGeometry(gmns.GeometryIDFromInt64(7), forward))
	assert.NoError(t, err)
	return network
}

func readExportRows(t *testing.T, directory, filename string) []map[string]string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(directory, filename))
	if err != nil {
		t.Fatal(err)
	}
	reader := csv.NewReader(bytes.NewReader(data))
	reader.Comma = ';'
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	rows := make([]map[string]string, 0, len(records)-1)
	for _, record := range records[1:] {
		row := make(map[string]string, len(records[0]))
		for index, column := range records[0] {
			row[column] = record[index]
		}
		rows = append(rows, row)
	}
	return rows
}

func TestSharedGeometryExportRoundTrip(t *testing.T) {
	network := geometryExportNetwork(t)
	// Distinct arcs with reversed endpoints must retain their own records.
	arc := orb.LineString{{37.01, 55.01}, {37.009, 55.001}, {37, 55}}
	network.Links[gmns.LinkIDFromInt64(22)] = macro.NewLinkFrom(gmns.LinkIDFromInt64(22), gmns.NodeIDFromInt64(2), gmns.NodeIDFromInt64(1), macro.WithLineGeom(arc), macro.WithGeometryID(gmns.GeometryIDFromInt64(8)), macro.WithDirFlag(gmns.DIR_FLAG_FORWARD))
	// Independent references are not merged even if the coordinates coincide.
	network.Links[gmns.LinkIDFromInt64(23)] = macro.NewLinkFrom(gmns.LinkIDFromInt64(23), gmns.NodeIDFromInt64(1), gmns.NodeIDFromInt64(2), macro.WithLineGeom(network.Links[gmns.LinkIDFromInt64(20)].Geom().Clone()), macro.WithGeometryID(gmns.GeometryIDFromInt64(9)), macro.WithDirFlag(gmns.DIR_FLAG_FORWARD))
	assert.NoError(t, network.AddGeometry(gmns.NewGeometry(gmns.GeometryIDFromInt64(8), arc)))
	assert.NoError(t, network.AddGeometry(gmns.NewGeometry(gmns.GeometryIDFromInt64(9), network.Links[gmns.LinkIDFromInt64(20)].Geom())))
	inlineDir := t.TempDir()
	sharedDir := t.TempDir()
	err := ExportToCSV(network, inlineDir)
	if err != nil {
		t.Fatal(err)
	}
	err = ExportToCSVWithGeometry(network, sharedDir, GEOMETRY_SHARED)
	if err != nil {
		t.Fatal(err)
	}
	geometries := readExportRows(t, sharedDir, "geometry.csv")
	if len(geometries) != 3 {
		t.Fatalf("got %d geometries, expected 3", len(geometries))
	}
	shapes := make(map[string]orb.LineString, len(geometries))
	for _, row := range geometries {
		shape, err := wkt.UnmarshalLineString(row["geometry"])
		if err != nil {
			t.Fatal(err)
		}
		shapes[row["geometry_id"]] = shape
	}
	inlineRows := readExportRows(t, inlineDir, "link.csv")
	sharedRows := readExportRows(t, sharedDir, "link.csv")
	if len(sharedRows) != len(inlineRows) {
		t.Fatal("link count changed")
	}
	for index, row := range sharedRows {
		_, hasInlineGeometry := row["geom"]
		if hasInlineGeometry {
			t.Fatal("shared export duplicates inline geometry")
		}
		shape, exists := shapes[row["geometry_id"]]
		if !exists {
			t.Fatalf("missing reference %s", row["geometry_id"])
		}
		shape = shape.Clone()
		switch row["dir_flag"] {
		case "1":
		case "-1":
			shape.Reverse()
		default:
			t.Fatalf("invalid direction %q", row["dir_flag"])
		}
		linkID, err := strconv.Atoi(row["id"])
		if err != nil {
			t.Fatal(err)
		}
		if !shape.Equal(network.Links[gmns.LinkIDFromInt64(int64(linkID))].Geom()) {
			t.Fatalf("link %d: reconstructed travel path differs", linkID)
		}
		delete(row, "geometry_id")
		delete(row, "dir_flag")
		delete(inlineRows[index], "geom")
		if !reflect.DeepEqual(row, inlineRows[index]) {
			t.Fatalf("link %d: nongeometry properties changed", linkID)
		}
	}
	if !reflect.DeepEqual(readExportRows(t, inlineDir, "node.csv"), readExportRows(t, sharedDir, "node.csv")) {
		t.Fatal("node export changed")
	}
	first, err := os.ReadFile(filepath.Join(sharedDir, "geometry.csv"))
	if err != nil {
		t.Fatal(err)
	}
	err = ExportToCSVWithGeometry(network, sharedDir, GEOMETRY_SHARED)
	if err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(filepath.Join(sharedDir, "geometry.csv"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("geometry export is not deterministic")
	}
}

func TestInlineGeometryCompatibility(t *testing.T) {
	network := geometryExportNetwork(t)
	defaultDir := t.TempDir()
	inlineDir := t.TempDir()
	err := ExportToCSV(network, defaultDir)
	if err != nil {
		t.Fatal(err)
	}
	err = ExportToCSVWithGeometry(network, inlineDir, GEOMETRY_INLINE)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"node.csv", "link.csv"} {
		original, err := os.ReadFile(filepath.Join(defaultDir, name))
		if err != nil {
			t.Fatal(err)
		}
		explicit, err := os.ReadFile(filepath.Join(inlineDir, name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(original, explicit) {
			t.Fatalf("default and inline differ for %s", name)
		}
	}
	_, err = os.Stat(filepath.Join(defaultDir, "geometry.csv"))
	if !os.IsNotExist(err) {
		t.Fatal("inline export unexpectedly created geometry.csv")
	}
	rows := readExportRows(t, defaultDir, "link.csv")
	for _, row := range rows {
		if row["geom"] == "" || len(row) != 19 {
			t.Fatal("legacy geometry or columns changed")
		}
	}
}

type invalidSharedGeometryCase struct {
	name    string
	mutate  func(*macro.Net)
	message string
}

func TestInvalidSharedGeometry(t *testing.T) {
	cases := []invalidSharedGeometryCase{
		{name: "missing reference", mutate: func(network *macro.Net) {
			network.Links[gmns.LinkIDFromInt64(20)] = macro.NewLinkFrom(gmns.LinkIDFromInt64(20), gmns.NodeIDFromInt64(1), gmns.NodeIDFromInt64(2))
		}, message: "geometry_id"},
		{name: "missing flag", mutate: func(network *macro.Net) {
			network.Links[gmns.LinkIDFromInt64(20)] = macro.NewLinkFrom(gmns.LinkIDFromInt64(20), gmns.NodeIDFromInt64(1), gmns.NodeIDFromInt64(2), macro.WithGeometryID(gmns.GeometryIDFromInt64(7)))
		}, message: "dir_flag"},
		{name: "zero flag", mutate: func(network *macro.Net) {
			macro.WithDirFlag(gmns.DIR_FLAG_NONE)(network.Links[gmns.LinkIDFromInt64(20)])
		}, message: "dir_flag"},
		{name: "invalid flag", mutate: func(network *macro.Net) { macro.WithDirFlag(2)(network.Links[gmns.LinkIDFromInt64(20)]) }, message: "dir_flag"},
		{name: "contradictory orientation", mutate: func(network *macro.Net) {
			macro.WithDirFlag(gmns.DIR_FLAG_FORWARD)(network.Links[gmns.LinkIDFromInt64(21)])
		}, message: "contradicts"},
		{name: "contradictory path", mutate: func(network *macro.Net) { network.Links[gmns.LinkIDFromInt64(21)].Geom()[1][0] += 0.001 }, message: "contradicts"},
		{name: "empty line", mutate: func(network *macro.Net) { macro.WithLineGeom(nil)(network.Links[gmns.LinkIDFromInt64(20)]) }, message: "at least two"},
		{name: "nonfinite point", mutate: func(network *macro.Net) { network.Links[gmns.LinkIDFromInt64(20)].Geom()[1][0] = math.NaN() }, message: "finite"},
		{name: "missing node", mutate: func(network *macro.Net) { delete(network.Nodes, gmns.NodeIDFromInt64(1)) }, message: "missing source or target"},
		{name: "wrong endpoints", mutate: func(network *macro.Net) { network.Links[gmns.LinkIDFromInt64(20)].Geom().Reverse() }, message: "source and target"},
		{name: "nil link", mutate: func(network *macro.Net) { network.Links[gmns.LinkIDFromInt64(20)] = nil }, message: "nil link"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			network := geometryExportNetwork(t)
			testCase.mutate(network)
			directory := t.TempDir()
			previous := []byte("previous contents")
			err := os.WriteFile(filepath.Join(directory, "link.csv"), previous, 0600)
			if err != nil {
				t.Fatal(err)
			}
			err = ExportToCSVWithGeometry(network, directory, GEOMETRY_SHARED)
			if err == nil || !strings.Contains(err.Error(), testCase.message) {
				t.Fatalf("expected %q, got %v", testCase.message, err)
			}
			actual, err := os.ReadFile(filepath.Join(directory, "link.csv"))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(previous, actual) {
				t.Fatal("validation failure overwrote existing output")
			}
		})
	}
}

func TestReverseOnlySharedGeometry(t *testing.T) {
	network := geometryExportNetwork(t)
	delete(network.Links, gmns.LinkIDFromInt64(20))
	directory := t.TempDir()
	err := ExportToCSVWithGeometry(network, directory, GEOMETRY_SHARED)
	if err != nil {
		t.Fatal(err)
	}
	rows := readExportRows(t, directory, "geometry.csv")
	shape, err := wkt.UnmarshalLineString(rows[0]["geometry"])
	if err != nil {
		t.Fatal(err)
	}
	shape.Reverse()
	if !shape.Equal(network.Links[gmns.LinkIDFromInt64(21)].Geom()) {
		t.Fatal("canonical shape depends on the presence of the forward link")
	}
}

func TestInvalidGeometryMode(t *testing.T) {
	directory := t.TempDir()
	err := ExportToCSVWithGeometry(geometryExportNetwork(t), directory, GeometryMode("unknown"))
	if err == nil {
		t.Fatal("invalid mode accepted")
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatal("invalid mode wrote files")
	}
}
