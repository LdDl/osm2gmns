package main

import (
	"bytes"
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

type commandCase struct {
	name     string
	levels   string
	geometry string
	expected []string
	absent   []string
}

func TestCLIStandardOutputs(t *testing.T) {
	cases := []commandCase{
		{
			name:     "all inline",
			levels:   "macro,meso,micro",
			geometry: "inline",
			expected: []string{"node_macro.csv", "link_macro.csv", "movement_macro.csv", "node_meso.csv", "link_micro.csv"},
			absent:   []string{"movement_meso.csv", "movement_micro.csv", "mesolink.csv"},
		},
		{
			name:     "macro shared",
			levels:   "macro",
			geometry: "shared",
			expected: []string{"geometry_macro.csv", "datapackage_macro.json"},
			absent:   []string{"node_meso.csv"},
		},
		{
			name:     "micro only",
			levels:   "micro",
			geometry: "inline",
			expected: []string{"node_micro.csv", "link_micro.csv", "config_micro.csv"},
			absent:   []string{"node_macro.csv", "node_meso.csv", "movement_macro.csv"},
		},
		{
			name:     "movement includes its references",
			levels:   "movement",
			geometry: "inline",
			expected: []string{"movement_macro.csv", "node_macro.csv", "link_macro.csv"},
			absent:   []string{"node_meso.csv", "node_micro.csv"},
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			directory := t.TempDir()
			args := []string{
				"-input", "../../sample.osm", "-output", directory,
				"-networks", testCase.levels, "-geometry", testCase.geometry,
				"-auto-persons", "1.6", "-verbose=false", "-suppress-warnings=true",
			}
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			err := run(args, &stdout, &stderr)
			assert.NoError(t, err)
			if err != nil {
				return
			}
			for _, filename := range testCase.expected {
				assert.FileExists(t, filepath.Join(directory, filename))
			}
			for _, filename := range testCase.absent {
				assert.NoFileExists(t, filepath.Join(directory, filename))
			}
			entries, err := os.ReadDir(directory)
			assert.NoError(t, err)
			for _, entry := range entries {
				assert.False(t, entry.IsDir(), "output stays flat")
				if entry.Name() == "use_definition_macro.csv" || entry.Name() == "use_definition_micro.csv" {
					file, openErr := os.Open(filepath.Join(directory, entry.Name()))
					assert.NoError(t, openErr)
					if openErr != nil {
						return
					}
					rows, readErr := csv.NewReader(file).ReadAll()
					assert.NoError(t, readErr)
					assert.NoError(t, file.Close())
					assert.Equal(t, "auto", rows[1][0])
					assert.Equal(t, "1.6", rows[1][1])
				}
			}
		})
	}
}

func TestCLIInvalidOptionsWriteNothing(t *testing.T) {
	directory := t.TempDir()
	args := []string{"-input", "../../sample.osm", "-output", directory, "-auto-pce", "-1"}
	var output bytes.Buffer
	assert.Error(t, run(args, &output, &output))
	entries, err := os.ReadDir(directory)
	assert.NoError(t, err)
	assert.Empty(t, entries)
	_, err = parseNetworkLevels("macro,typo")
	assert.Error(t, err)
	_, err = parseAgentTypes("auto,typo")
	assert.Error(t, err)
}
