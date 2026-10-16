package csvimport

import (
	"bytes"
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LdDl/go-gmns/generators"
	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/osm2gmns/csvexport"
	"github.com/LdDl/osm2gmns/osmmacro"
	"github.com/LdDl/osm2gmns/osmmeso"
	"github.com/stretchr/testify/assert"
)

func TestDetailedRecordsRoundTripAndReorderedPreparation(t *testing.T) {
	for _, kind := range []gmns.IDType{gmns.IDTypeString, gmns.IDTypeInteger} {
		t.Run(string(kind), func(t *testing.T) {
			var expected csvexport.Dataset
			for _, shared := range []bool{false, true} {
				files := standardFiles(t, kind, ';', shared, false)
				source, err := ReadMacro(files, "network.json")
				assert.NoError(t, err)
				if err != nil {
					return
				}
				linkID, err := gmns.ParseLinkID("13", kind)
				assert.NoError(t, err)
				link := source.Links[linkID]
				laneID, err := gmns.ParseLaneID("001", kind)
				assert.NoError(t, err)
				lane := gmns.NewLane(laneID, linkID, 3, gmns.WithLaneWidthMeters(3.5), gmns.WithLaneAllowedUses("auto"))
				err = source.AddLane(lane)
				assert.NoError(t, err)
				segmentID, err := gmns.ParseSegmentID("900719925474099312345", kind)
				assert.NoError(t, err)
				segment := gmns.NewSegment(segmentID, linkID, link.TargetNode(), 48, 88,
					gmns.WithSegmentLanesNum(2), gmns.WithSegmentRightLanesAdded(-1),
					gmns.WithSegmentLeftLanesAdded(0), gmns.WithSegmentCapacity(0),
					gmns.WithSegmentFreeSpeed(25.5), gmns.WithSegmentGrade(-2),
					gmns.WithSegmentToll(0), gmns.WithSegmentRightOfWayWidthMeters(10),
					gmns.WithSegmentBikeFacility("none"), gmns.WithSegmentPedFacility("none"),
					gmns.WithSegmentParking("none"), gmns.WithSegmentAllowedUses("auto"), gmns.WithSegmentJurisdiction("city"))
				err = source.AddSegment(segment)
				assert.NoError(t, err)
				detailID, err := gmns.ParseSegmentLaneID("001", kind)
				assert.NoError(t, err)
				detail := gmns.NewSegmentLane(detailID, segmentID, 0, gmns.WithSegmentLaneParentLane(laneID))
				err = source.AddSegmentLane(detail)
				assert.NoError(t, err)
				before, err := osmmacro.BuildDataset(source, osmmacro.GEOMETRY_INLINE)
				assert.NoError(t, err)
				first, err := generators.PrepareMesoscopic(source, generators.WithDetailedLaneLayout())
				assert.NoError(t, err)
				if err != nil {
					return
				}
				firstExport, err := osmmeso.BuildDataset(first.Meso())
				assert.NoError(t, err)
				if !shared {
					expected = firstExport
				} else {
					assert.Equal(t, expected, firstExport)
				}
				directory := t.TempDir()
				err = osmmacro.ExportToCSV(first.Macro(), directory, csvexport.WithDelimiter(','))
				assert.NoError(t, err)
				reverseCSVFiles(t, directory, ',')
				restored, err := ReadMacro(os.DirFS(directory), "datapackage_macro.json")
				assert.NoError(t, err)
				if err != nil {
					return
				}
				assert.Equal(t, source.LaneRecords(), restored.LaneRecords())
				assert.Equal(t, source.SegmentRecords(), restored.SegmentRecords())
				assert.Equal(t, source.SegmentLaneRecords(), restored.SegmentLaneRecords())
				after, err := osmmacro.BuildDataset(restored, osmmacro.GEOMETRY_INLINE)
				assert.NoError(t, err)
				assert.Equal(t, before.Tables, after.Tables)
				rebuilt, err := generators.PrepareMesoscopic(restored, generators.WithDetailedLaneLayout())
				assert.NoError(t, err)
				if err != nil {
					return
				}
				actual, err := osmmeso.BuildDataset(rebuilt.Meso())
				assert.NoError(t, err)
				assert.Equal(t, expected, actual)
				assert.Equal(t, first.Sections(), rebuilt.Sections())
				assert.Equal(t, first.Connections(), rebuilt.Connections())
			}
		})
	}
}

func reverseCSVFiles(t *testing.T, directory string, delimiter rune) {
	t.Helper()
	entries, err := os.ReadDir(directory)
	assert.NoError(t, err)
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".csv") {
			continue
		}
		filename := filepath.Join(directory, entry.Name())
		data, readErr := os.ReadFile(filename)
		assert.NoError(t, readErr)
		reader := csv.NewReader(bytes.NewReader(data))
		reader.Comma = delimiter
		rows, readErr := reader.ReadAll()
		assert.NoError(t, readErr)
		for first, last := 1, len(rows)-1; first < last; first, last = first+1, last-1 {
			rows[first], rows[last] = rows[last], rows[first]
		}
		for _, row := range rows {
			for first, last := 0, len(row)-1; first < last; first, last = first+1, last-1 {
				row[first], row[last] = row[last], row[first]
			}
		}
		var buffer bytes.Buffer
		writer := csv.NewWriter(&buffer)
		writer.Comma = delimiter
		writeErr := writer.WriteAll(rows)
		assert.NoError(t, writeErr)
		writeErr = os.WriteFile(filename, buffer.Bytes(), 0600)
		assert.NoError(t, writeErr)
	}
}
