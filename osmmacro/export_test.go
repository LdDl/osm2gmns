package osmmacro

import (
	"bytes"
	"encoding/csv"
	"math"
	"os"
	"path/filepath"
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
	assert.NoError(t, network.SetConfig(gmns.NewConfig(gmns.IDTypeInteger)))
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
	arc := orb.LineString{{37.01, 55.01}, {37.009, 55.001}, {37, 55}}
	arcID := gmns.GeometryIDFromInt64(8)
	network.Links[gmns.LinkIDFromInt64(22)] = macro.NewLinkFrom(gmns.LinkIDFromInt64(22), gmns.NodeIDFromInt64(2), gmns.NodeIDFromInt64(1),
		macro.WithLineGeom(arc),
		macro.WithGeometryID(arcID),
		macro.WithDirFlag(gmns.DIR_FLAG_FORWARD),
	)
	assert.NoError(t, network.AddGeometry(gmns.NewGeometry(arcID, arc)))
	inlineDir := t.TempDir()
	sharedDir := t.TempDir()
	assert.NoError(t, ExportToCSV(network, inlineDir))
	assert.NoError(t, ExportToCSVWithGeometry(network, sharedDir, GEOMETRY_SHARED))
	geometries := readExportRows(t, sharedDir, "geometry_macro.csv")
	assert.Len(t, geometries, 2, "opposite road directions share a shape; a different arc does not")
	shapes := make(map[string]orb.LineString)
	for _, row := range geometries {
		shape, err := wkt.UnmarshalLineString(row["geometry"])
		assert.NoError(t, err)
		shapes[row["geometry_id"]] = shape
	}
	inlineRows := readExportRows(t, inlineDir, "link_macro.csv")
	sharedRows := readExportRows(t, sharedDir, "link_macro.csv")
	assert.Len(t, sharedRows, len(inlineRows))
	for index, row := range sharedRows {
		assert.Empty(t, row["geometry"])
		shape := shapes[row["geometry_id"]].Clone()
		if row["dir_flag"] == "-1" {
			shape.Reverse()
		}
		expected, err := wkt.UnmarshalLineString(inlineRows[index]["geometry"])
		assert.NoError(t, err)
		assert.Equal(t, expected, shape)
		for _, field := range []string{"geometry_id", "geometry", "dir_flag"} {
			delete(row, field)
			delete(inlineRows[index], field)
		}
		assert.Equal(t, inlineRows[index], row)
	}
	assert.Equal(t, readExportRows(t, inlineDir, "node_macro.csv"), readExportRows(t, sharedDir, "node_macro.csv"))
}

func TestReverseOnlySharedGeometry(t *testing.T) {
	network := geometryExportNetwork(t)
	delete(network.Links, gmns.LinkIDFromInt64(20))
	directory := t.TempDir()
	assert.NoError(t, ExportToCSVWithGeometry(network, directory, GEOMETRY_SHARED))
	rows := readExportRows(t, directory, "geometry_macro.csv")
	shape, err := wkt.UnmarshalLineString(rows[0]["geometry"])
	assert.NoError(t, err)
	shape.Reverse()
	assert.Equal(t, network.Links[gmns.LinkIDFromInt64(21)].Geom(), shape)
}

func TestInvalidSharedGeometry(t *testing.T) {
	cases := []string{
		"missing reference", "missing flag", "zero flag", "invalid flag", "contradictory orientation",
		"contradictory path", "empty line", "nonfinite point", "missing node", "wrong endpoints", "nil link",
	}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			network := geometryExportNetwork(t)
			link := network.Links[gmns.LinkIDFromInt64(20)]
			switch name {
			case "missing reference":
				macro.WithoutGeometryID()(link)
			case "missing flag":
				macro.WithoutDirFlag()(link)
			case "zero flag":
				macro.WithDirFlag(gmns.DIR_FLAG_NONE)(link)
			case "invalid flag":
				macro.WithDirFlag(2)(link)
			case "contradictory orientation":
				macro.WithDirFlag(gmns.DIR_FLAG_REVERSE)(link)
			case "contradictory path":
				link.Geom()[1][0] += 0.001
			case "empty line":
				macro.WithLineGeom(nil)(link)
			case "nonfinite point":
				link.Geom()[1][0] = math.NaN()
			case "missing node":
				delete(network.Nodes, link.SourceNode())
			case "wrong endpoints":
				link.Geom().Reverse()
			case "nil link":
				network.Links[link.ID] = nil
			}
			directory := t.TempDir()
			previous := []byte("previous contents")
			filename := filepath.Join(directory, "link_macro.csv")
			assert.NoError(t, os.WriteFile(filename, previous, 0600))
			assert.Error(t, ExportToCSVWithGeometry(network, directory, GEOMETRY_SHARED))
			actual, err := os.ReadFile(filename)
			assert.NoError(t, err)
			assert.Equal(t, previous, actual)
		})
	}
}

func TestInvalidGeometryMode(t *testing.T) {
	directory := t.TempDir()
	assert.Error(t, ExportToCSVWithGeometry(geometryExportNetwork(t), directory, GeometryMode("unknown")))
	entries, err := os.ReadDir(directory)
	assert.NoError(t, err)
	assert.Empty(t, entries)
}
