package csvimport

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/LdDl/go-gmns/generators"
	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/osm2gmns/csvexport"
	"github.com/LdDl/osm2gmns/osmmacro"
	"github.com/LdDl/osm2gmns/osmmeso"
	"github.com/stretchr/testify/assert"
)

func standardFiles(t *testing.T, kind gmns.IDType, delimiter rune, shared, reversed bool) fstest.MapFS {
	t.Helper()
	tables := map[string][][]string{
		"config": {
			{"dataset_name", "short_length", "long_length", "speed", "crs", "geometry_field_format", "id_type", "currency"},
			{"sample", "meter", "meter", "km/h", "EPSG:4326", "WKT", string(kind), "RUB"},
		},
		"node": {
			{"node_id", "x_coord", "y_coord", "z_coord", "zone_id", "ctrl_type"},
			{"001", "37", "55", "0", "-1", ""},
			{"2", "37.002", "55", "", "", "signal"},
			{"900719925474099312345", "37.004", "55", "", "", ""},
		},
		"link": {
			{"link_id", "from_node_id", "to_node_id", "directed", "length", "lanes", "geometry", "geometry_id", "dir_flag", "capacity", "grade", "toll", "name", "allowed_uses"},
			{"13", "001", "2", "true", "128", "3", "LINESTRING(37 55,37.002 55)", "", "1", "1200.5", "0", "0", "Road; comma, quote\"\nsecond line", "motor"},
			{"14", "2", "900719925474099312345", "true", "128", "3", "LINESTRING(37.002 55,37.004 55)", "", "1", "", "", "", "", "auto"},
		},
		"movement": {
			{"mvmt_id", "node_id", "ib_link_id", "ob_link_id", "start_ib_lane", "end_ib_lane", "start_ob_lane", "end_ob_lane", "penalty", "capacity", "ctrl_type", "type", "allowed_uses"},
			{"49", "2", "13", "14", "3", "3", "2", "2", "0", "", "stop", "thru", "auto"},
		},
		"zone": {
			{"zone_id", "name", "boundary", "super_zone"},
			{"-1", "district", "", ""},
		},
		"use_definition": {
			{"use", "persons_per_vehicle", "pce", "description", "special_conditions"},
			{"auto", "1", "1", "cars", ""},
		},
		"use_group": {
			{"use_group", "uses", "description"},
			{"motor", "auto", "motor traffic"},
		},
	}
	if shared {
		tables["geometry"] = [][]string{
			{"geometry_id", "geometry"},
			{"0", "LINESTRING(37.002 55,37 55)"},
			{"1", "LINESTRING(37.004 55,37.002 55)"},
		}
		tables["link"][1][6] = ""
		tables["link"][1][7] = "0"
		tables["link"][1][8] = "-1"
		tables["link"][2][6] = ""
		tables["link"][2][7] = "1"
		tables["link"][2][8] = "-1"
	}
	files := fstest.MapFS{}
	resources := make([]map[string]any, 0, len(tables))
	// Descriptor order deliberately differs from dependency order.
	for _, name := range []string{"movement", "link", "node", "use_group", "use_definition", "zone", "geometry", "config"} {
		rows := tables[name]
		if rows == nil {
			continue
		}
		if reversed {
			for first, last := 1, len(rows)-1; first < last; first, last = first+1, last-1 {
				rows[first], rows[last] = rows[last], rows[first]
			}
			for _, cells := range rows {
				for first, last := 0, len(cells)-1; first < last; first, last = first+1, last-1 {
					cells[first], cells[last] = cells[last], cells[first]
				}
			}
		}
		var buffer bytes.Buffer
		writer := csv.NewWriter(&buffer)
		writer.Comma = delimiter
		err := writer.WriteAll(rows)
		assert.NoError(t, err)
		filename := "records/" + name + ".csv"
		files[filename] = &fstest.MapFile{Data: buffer.Bytes()}
		resources = append(resources, map[string]any{
			"name":    name,
			"path":    filename,
			"dialect": map[string]any{"delimiter": string(delimiter)},
		})
	}
	data, err := json.Marshal(map[string]any{"resources": resources})
	assert.NoError(t, err)
	files["network.json"] = &fstest.MapFile{Data: data}
	return files
}

func TestReadPrepareAndSaveStandardMacro(t *testing.T) {
	for _, kind := range []gmns.IDType{gmns.IDTypeInteger, gmns.IDTypeString} {
		t.Run(string(kind), func(t *testing.T) {
			files := standardFiles(t, kind, ';', false, false)
			source, err := ReadMacro(files, "network.json")
			assert.NoError(t, err)
			if err != nil {
				return
			}
			before, err := osmmacro.BuildDataset(source, osmmacro.GEOMETRY_INLINE)
			assert.NoError(t, err)
			prepared, err := generators.PrepareMesoscopic(source, generators.WithUniformLaneLayout())
			assert.NoError(t, err)
			if err != nil {
				return
			}
			expected, err := osmmeso.BuildDataset(prepared.Meso())
			assert.NoError(t, err)
			completed, err := osmmacro.BuildDataset(prepared.Macro(), osmmacro.GEOMETRY_INLINE)
			assert.NoError(t, err)
			assert.Equal(t, before.Tables, completed.Tables)
			for _, separator := range []rune{';', ',', '\t', '§'} {
				for _, shared := range []bool{false, true} {
					permuted, readErr := ReadMacro(standardFiles(t, kind, separator, shared, true), "network.json")
					assert.NoError(t, readErr)
					if readErr != nil {
						continue
					}
					result, prepareErr := generators.PrepareMesoscopic(permuted, generators.WithUniformLaneLayout())
					assert.NoError(t, prepareErr)
					if prepareErr != nil {
						continue
					}
					actual, buildErr := osmmeso.BuildDataset(result.Meso())
					assert.NoError(t, buildErr)
					assert.Equal(t, expected, actual)
				}
			}
			after, err := osmmacro.BuildDataset(source, osmmacro.GEOMETRY_INLINE)
			assert.NoError(t, err)
			assert.Equal(t, before, after)
			assert.Equal(t, standardFiles(t, kind, ';', false, false), files)
			directory := t.TempDir()
			err = csvexport.WriteDatasets(directory, []csvexport.Dataset{before, expected})
			assert.NoError(t, err)
			reread, err := ReadMacro(os.DirFS(directory), "datapackage_macro.json")
			assert.NoError(t, err)
			if err != nil {
				return
			}
			roundTrip, err := osmmacro.BuildDataset(reread, osmmacro.GEOMETRY_INLINE)
			assert.NoError(t, err)
			assert.Equal(t, before.Tables, roundTrip.Tables)
		})
	}
}

type importFailureCase struct {
	name        string
	file        string
	old         string
	replacement string
	message     string
}

func TestReadMacroReportsUnsupportedOrInvalidInput(t *testing.T) {
	cases := []importFailureCase{
		{
			name:        "unknown column",
			file:        "records/link.csv",
			old:         "lanes",
			replacement: "private_lanes",
			message:     "unsupported column",
		},
		{
			name:        "duplicate header",
			file:        "records/link.csv",
			old:         "name",
			replacement: "link_id",
			message:     "duplicate column",
		},
		{
			name:        "unsupported units",
			file:        "records/config.csv",
			old:         "meter",
			replacement: "mile",
			message:     "only meter",
		},
		{
			name:        "large integer never rounded",
			file:        "records/node.csv",
			old:         "900719925474099312345",
			replacement: "900719925474099312346",
			message:     "900719925474099312345",
		},
		{
			name:        "bad resource path",
			file:        "network.json",
			old:         "records/link.csv",
			replacement: "../link.csv",
			message:     "local relative CSV path",
		},
		{
			name:        "unsupported detail table",
			file:        "network.json",
			old:         "\"name\":\"movement\"",
			replacement: "\"name\":\"lane\"",
			message:     "resource \"lane\"",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			files := standardFiles(t, gmns.IDTypeInteger, ';', false, false)
			file := files[testCase.file]
			file.Data = []byte(strings.Replace(string(file.Data), testCase.old, testCase.replacement, 1))
			result, err := ReadMacro(files, "network.json")
			assert.Nil(t, result)
			assert.ErrorContains(t, err, testCase.message)
		})
	}
}
