package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/go-gmns/macro"
	"github.com/LdDl/osm2gmns/csvexport"
	"github.com/LdDl/osm2gmns/csvimport"
	"github.com/LdDl/osm2gmns/osmmacro"
	"github.com/paulmach/orb"
	"github.com/stretchr/testify/assert"
)

func runnerSource(t *testing.T, directory string) {
	t.Helper()
	source := macro.NewNet()
	err := source.SetConfig(gmns.NewConfig(gmns.IDTypeInteger))
	assert.NoError(t, err)
	first := gmns.NodeIDFromInt64(1)
	last := gmns.NodeIDFromInt64(2)
	shape := orb.LineString{{37, 55}, {37.002, 55}}
	source.Nodes[first] = macro.NewNodeFrom(first, macro.WithPointGeom(shape[0]))
	source.Nodes[last] = macro.NewNodeFrom(last, macro.WithPointGeom(shape[1]))
	linkID := gmns.LinkIDFromInt64(13)
	source.Links[linkID] = macro.NewLinkFrom(linkID, first, last,
		macro.WithLineGeom(shape), macro.WithLengthMeters(128), macro.WithLanesNum(3),
	)
	err = osmmacro.ExportToCSV(source, directory, csvexport.WithDelimiter(','))
	assert.NoError(t, err)
}

func TestRunnerExportsCompletedLanesAndSeparateAssumptions(t *testing.T) {
	directory := t.TempDir()
	input := filepath.Join(directory, "source")
	runnerSource(t, input)
	source, err := csvimport.ReadMacro(os.DirFS(input), "datapackage_macro.json")
	assert.NoError(t, err)
	if err != nil {
		return
	}
	macro.WithoutLanesNum()(source.Links[gmns.LinkIDFromInt64(13)])
	err = osmmacro.ExportToCSV(source, input)
	assert.NoError(t, err)
	before := runnerFiles(t, input)
	var output bytes.Buffer
	destination := filepath.Join(directory, "result")
	arguments := []string{"-input", filepath.Join(input, "datapackage_macro.json"), "-output", destination, "-lane-layout", "uniform"}
	err = run(arguments, &output)
	assert.ErrorContains(t, err, "link 13 lanes")
	_, statErr := os.Stat(destination)
	assert.True(t, os.IsNotExist(statErr))
	err = run(append(arguments, "-default-lanes", "4"), &output)
	assert.NoError(t, err)
	assert.Equal(t, before, runnerFiles(t, input))
	completed, err := csvimport.ReadMacro(os.DirFS(destination), "datapackage_macro.json")
	assert.NoError(t, err)
	if err != nil {
		return
	}
	lanes, present := completed.Links[gmns.LinkIDFromInt64(13)].LanesNum()
	assert.True(t, present)
	assert.Equal(t, 4, lanes)
	for _, name := range []string{"provenance_macro.json", "provenance_meso.json"} {
		data, readErr := os.ReadFile(filepath.Join(destination, name))
		assert.NoError(t, readErr)
		var metadata map[string]any
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.UseNumber()
		err = decoder.Decode(&metadata)
		assert.NoError(t, err)
		preparation := metadata["preparation"].(map[string]any)
		assert.Equal(t, "constant", preparation["missing_lanes_policy"])
		assert.Equal(t, json.Number("4"), preparation["default_lanes"])
		assumptions := preparation["assumptions"].([]any)
		assert.Len(t, assumptions, 1)
		assert.Equal(t, map[string]any{
			"link_id": json.Number("13"),
			"field":   "lanes",
			"value":   json.Number("4"),
			"reason":  "explicit-default-for-absent-value",
		}, assumptions[0])
	}
	invalid := filepath.Join(directory, "invalid")
	err = run([]string{"-input", filepath.Join(input, "datapackage_macro.json"), "-output", invalid,
		"-lane-layout", "uniform", "-default-lanes", "0"}, &output)
	assert.ErrorContains(t, err, "default lanes must be in 1..99")
	_, statErr = os.Stat(invalid)
	assert.True(t, os.IsNotExist(statErr))
}

func TestRunnerProducesReproducibleDatasets(t *testing.T) {
	directory := t.TempDir()
	input := filepath.Join(directory, "source")
	runnerSource(t, input)
	descriptor := filepath.Join(input, "datapackage_macro.json")
	before := runnerFiles(t, input)
	first := filepath.Join(directory, "first")
	second := filepath.Join(directory, "second")
	var output bytes.Buffer
	for _, destination := range []string{first, second} {
		err := run([]string{"-input", descriptor, "-output", destination, "-lane-layout", "uniform"}, &output)
		assert.NoError(t, err)
	}
	assert.Contains(t, output.String(), "Prepared meso: 2 nodes, 1 links")
	assert.Equal(t, runnerFiles(t, first), runnerFiles(t, second))
	assert.Equal(t, before, runnerFiles(t, input))
	read, err := csvimport.ReadMacro(os.DirFS(first), "datapackage_macro.json")
	assert.NoError(t, err)
	assert.Len(t, read.Links, 1)
	err = run([]string{"-input", descriptor, "-output", input, "-lane-layout", "uniform"}, &output)
	assert.ErrorContains(t, err, "already exists")
	assert.Equal(t, before, runnerFiles(t, input))
	failed := filepath.Join(directory, "failed")
	err = run([]string{"-input", descriptor, "-output", failed}, &output)
	assert.ErrorContains(t, err, "explicitly accept")
	_, err = os.Stat(failed)
	assert.True(t, os.IsNotExist(err))
}

func runnerFiles(t *testing.T, directory string) map[string][]byte {
	t.Helper()
	entries, err := os.ReadDir(directory)
	assert.NoError(t, err)
	result := make(map[string][]byte)
	for _, entry := range entries {
		data, readErr := os.ReadFile(filepath.Join(directory, entry.Name()))
		assert.NoError(t, readErr)
		result[entry.Name()] = data
	}
	return result
}

func TestRunnerDetailedSectionsAndLineage(t *testing.T) {
	directory := t.TempDir()
	input := filepath.Join(directory, "source")
	runnerSource(t, input)
	source, err := csvimport.ReadMacro(os.DirFS(input), "datapackage_macro.json")
	assert.NoError(t, err)
	if err != nil {
		return
	}
	link := source.Links[gmns.LinkIDFromInt64(13)]
	lane := gmns.NewLane(gmns.LaneIDFromInt64(3), link.ID, 3)
	err = source.AddLane(lane)
	assert.NoError(t, err)
	segment := gmns.NewSegment(gmns.SegmentIDFromInt64(1), link.ID, link.SourceNode(), 40, 80)
	err = source.AddSegment(segment)
	assert.NoError(t, err)
	detail := gmns.NewSegmentLane(gmns.SegmentLaneIDFromInt64(1), segment.ID, 0, gmns.WithSegmentLaneParentLane(lane.ID))
	err = source.AddSegmentLane(detail)
	assert.NoError(t, err)
	err = osmmacro.ExportToCSV(source, input)
	assert.NoError(t, err)
	before := runnerFiles(t, input)
	destination := filepath.Join(directory, "result")
	var output bytes.Buffer
	arguments := []string{"-input", filepath.Join(input, "datapackage_macro.json"), "-output", destination, "-lane-layout", "uniform"}
	err = run(arguments, &output)
	assert.ErrorContains(t, err, "lane 3: uniform layout")
	_, statErr := os.Stat(destination)
	assert.True(t, os.IsNotExist(statErr))
	arguments[len(arguments)-1] = "detailed"
	err = run(arguments, &output)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	assert.Equal(t, before, runnerFiles(t, input))
	restored, err := csvimport.ReadMacro(os.DirFS(destination), "datapackage_macro.json")
	assert.NoError(t, err)
	assert.Equal(t, source.SegmentLaneRecords(), restored.SegmentLaneRecords())
	data, err := os.ReadFile(filepath.Join(destination, "provenance_meso.json"))
	assert.NoError(t, err)
	var metadata map[string]any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	err = decoder.Decode(&metadata)
	assert.NoError(t, err)
	preparation := metadata["preparation"].(map[string]any)
	assert.Equal(t, "detailed", preparation["lane_layout"])
	sections := preparation["sections"].([]any)
	assert.Len(t, sections, 3)
	middle := sections[1].(map[string]any)
	assert.Equal(t, json.Number("1"), middle["source_segment_id"])
	assert.Equal(t, []any{json.Number("1"), json.Number("2")}, middle["lane_numbers"])
	second := filepath.Join(directory, "second")
	arguments[3] = second
	err = run(arguments, &output)
	assert.NoError(t, err)
	assert.Equal(t, runnerFiles(t, destination), runnerFiles(t, second))
}
