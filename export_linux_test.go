package osm2gmns_test

import (
	"encoding/csv"
	"github.com/LdDl/go-gmns/gmns"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/LdDl/go-gmns/meso"
	"github.com/LdDl/go-gmns/micro"
	"github.com/LdDl/go-gmns/movement"
	"github.com/LdDl/osm2gmns/osmmeso"
	"github.com/LdDl/osm2gmns/osmmicro"
	"github.com/LdDl/osm2gmns/osmmovement"
	"github.com/stretchr/testify/assert"
)

type csvExportCase struct {
	name     string
	filename string
	export   func(string) error
}

func csvExportCases() []csvExportCase {
	mesoNet := meso.NewNet()
	mesoNet.Nodes[gmns.NodeIDFromInt64(1)] = meso.NewNodeFrom(gmns.NodeIDFromInt64(1))
	mesoNet.Links[gmns.LinkIDFromInt64(2)] = meso.NewLinkFrom(gmns.LinkIDFromInt64(2), gmns.NodeIDFromInt64(1), gmns.NodeIDFromInt64(1))
	microNet := micro.NewNet()
	microNet.AddNode(micro.NewNodeFrom(gmns.NodeIDFromInt64(1)))
	microNet.AddLink(micro.NewLinkFrom(gmns.LinkIDFromInt64(2), gmns.NodeIDFromInt64(1), gmns.NodeIDFromInt64(1)))
	movements := movement.NewMovementsStorage()
	movements[gmns.MovementIDFromInt64(3)] = movement.NewMovement(gmns.MovementIDFromInt64(3), gmns.NodeIDFromInt64(1), gmns.LinkIDFromInt64(2), gmns.LinkIDFromInt64(2), movement.MOVEMENT_SBT, movement.MOVEMENT_TYPE_THRU)
	return []csvExportCase{
		{
			name:     "movement",
			filename: "movement.csv",
			export: func(directory string) error {
				return osmmovement.ExportToCSV(movements, filepath.Join(directory, "movement.csv"))
			},
		},
		{
			name:     "meso nodes",
			filename: "mesonode.csv",
			export: func(directory string) error {
				return osmmeso.ExportToCSV(mesoNet, directory)
			},
		},
		{
			name:     "meso links",
			filename: "mesolink.csv",
			export: func(directory string) error {
				return osmmeso.ExportToCSV(mesoNet, directory)
			},
		},
		{
			name:     "micro nodes",
			filename: "micronode.csv",
			export: func(directory string) error {
				return osmmicro.ExportToCSV(microNet, directory)
			},
		},
		{
			name:     "micro links",
			filename: "microlink.csv",
			export: func(directory string) error {
				return osmmicro.ExportToCSV(microNet, directory)
			},
		},
	}
}

func TestCSVFlushErrors(t *testing.T) {
	_, err := os.Stat("/dev/full")
	if err != nil {
		t.Skip("requires /dev/full to simulate a full disk")
	}
	for _, testCase := range csvExportCases() {
		t.Run(testCase.name, func(t *testing.T) {
			directory := t.TempDir()
			err := os.Symlink("/dev/full", filepath.Join(directory, testCase.filename))
			assert.NoError(t, err)
			if err != nil {
				return
			}
			err = testCase.export(directory)
			assert.ErrorIs(t, err, syscall.ENOSPC)
		})
	}
}

func TestCSVFilesComplete(t *testing.T) {
	for _, testCase := range csvExportCases() {
		t.Run(testCase.name, func(t *testing.T) {
			directory := t.TempDir()
			err := testCase.export(directory)
			assert.NoError(t, err)
			if err != nil {
				return
			}
			file, err := os.Open(filepath.Join(directory, testCase.filename))
			assert.NoError(t, err)
			if err != nil {
				return
			}
			defer file.Close()
			reader := csv.NewReader(file)
			reader.Comma = ';'
			rows, err := reader.ReadAll()
			assert.NoError(t, err)
			if err != nil {
				return
			}
			assert.Len(t, rows, 2, "expected header and one record")
		})
	}
}
