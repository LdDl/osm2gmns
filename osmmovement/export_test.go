package osmmovement

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/go-gmns/gmns/types"
	"github.com/LdDl/go-gmns/movement"
	"github.com/stretchr/testify/assert"
)

func TestStandaloneExportRejectsUnpreparedUsesAndMissingMarker(t *testing.T) {
	directory := t.TempDir()
	filename := filepath.Join(directory, "movement.csv")
	original := []byte("existing table")
	assert.NoError(t, os.WriteFile(filename, original, 0600))
	identifier := gmns.MovementIDFromString("move")
	node := gmns.NodeIDFromString("node")
	link := gmns.LinkIDFromString("link")
	agents := []types.AgentType{types.AGENT_AUTO}
	record := movement.NewMovement(identifier, node, link, link, "NBL", "left",
		movement.WithAllowedAgentTypes(agents),
	)
	records := movement.MovementsStorage{
		identifier: record,
	}
	assert.Error(t, ExportToCSV(records, filename))
	movement.WithAllowedUses("auto")(record)
	record.ID = gmns.MovementIDFromString("NaN")
	delete(records, identifier)
	records[record.ID] = record
	assert.Error(t, ExportToCSV(records, filename))
	actual, err := os.ReadFile(filename)
	assert.NoError(t, err)
	assert.Equal(t, original, actual)
}
