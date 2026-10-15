package csvexport

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func smallDataset() Dataset {
	config := Table{
		Name:   "config",
		Header: []string{"id_type"},
		Rows: [][]string{
			{"integer"},
		},
	}
	nodes := Table{
		Name:   "node",
		Header: []string{"node_id", "x_coord", "y_coord"},
		Rows: [][]string{
			{"0", "0", "0"},
		},
	}
	links := Table{
		Name:   "link",
		Header: []string{"link_id", "from_node_id", "to_node_id", "directed"},
	}
	return Dataset{
		Level:  "macro",
		Tables: []Table{config, nodes, links},
	}
}

func TestBatchValidationBeforeReplacement(t *testing.T) {
	directory := t.TempDir()
	original := []byte("keep me")
	filename := filepath.Join(directory, "node_macro.csv")
	assert.NoError(t, os.WriteFile(filename, original, 0600))
	valid := smallDataset()
	invalid := smallDataset()
	invalid.Level = "meso"
	invalid.Tables[1].Rows[0] = []string{"wrong width"}
	assert.Error(t, WriteDatasets(directory, []Dataset{valid, invalid}))
	actual, err := os.ReadFile(filename)
	assert.NoError(t, err)
	assert.Equal(t, original, actual)
	assert.NoFileExists(t, filepath.Join(directory, "config_macro.csv"))
}

func TestStagingFailureKeepsPreviousDataset(t *testing.T) {
	directory := t.TempDir()
	filename := filepath.Join(directory, "node_macro.csv")
	original := []byte("previous dataset")
	assert.NoError(t, os.WriteFile(filename, original, 0600))
	dataset := smallDataset()
	dataset.Provenance = map[string]any{"unsupported": make(chan int)}
	assert.Error(t, WriteDatasets(directory, []Dataset{dataset}))
	actual, err := os.ReadFile(filename)
	assert.NoError(t, err)
	assert.Equal(t, original, actual)
	entries, err := os.ReadDir(directory)
	assert.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestReplacementRemovesOnlyManagedStaleTables(t *testing.T) {
	directory := t.TempDir()
	stale := filepath.Join(directory, "zone_macro.csv")
	otherLevel := filepath.Join(directory, "node_meso.csv")
	unrelated := filepath.Join(directory, "notes.txt")
	for _, filename := range []string{stale, otherLevel, unrelated} {
		assert.NoError(t, os.WriteFile(filename, []byte("keep"), 0600))
	}
	assert.NoError(t, WriteDatasets(directory, []Dataset{smallDataset()}))
	assert.NoFileExists(t, stale)
	assert.FileExists(t, otherLevel)
	assert.FileExists(t, unrelated)
	nodes, err := os.ReadFile(filepath.Join(directory, "node_macro.csv"))
	assert.NoError(t, err)
	assert.Equal(t, "node_id;x_coord;y_coord\n0;0;0\n", string(nodes))
}

func TestNonRegularDestinationIsNotReplaced(t *testing.T) {
	directory := t.TempDir()
	blocker := filepath.Join(directory, "node_macro.csv")
	assert.NoError(t, os.Mkdir(blocker, 0755))
	assert.Error(t, WriteDatasets(directory, []Dataset{smallDataset()}))
	assert.DirExists(t, blocker)
	assert.NoFileExists(t, filepath.Join(directory, "config_macro.csv"))
}
