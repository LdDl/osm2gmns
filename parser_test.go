package osm2gmns

import (
	"testing"

	"github.com/LdDl/osm2gmns/types"
)

func TestParser(t *testing.T) {
	parser := NewParser(
		"./sample.osm",
		WithPreparePOI(false),
		WithStrictMode(false),
		WithVerbose(true),
		WithAllowedAgentTypes(types.AGENT_TYPES_DEFAULT),
	)
	t.Log(parser)

	osmData, err := parser.ReadOSM()
	if err != nil {
		t.Error(err)
		return
	}

	macroNet, err := osmData.GenerateMacroscopic(parser.preparePOI)
	if err != nil {
		t.Error(err)
		return
	}
	movements, err := macroNet.GenerateMovements()
	if err != nil {
		t.Error(err)
		return
	}
	_ = movements

	macroNet.ExportToCSV("test_data/test.csv")
	movements.ExportToCSV("test_data/test_movement.csv")
	// @todo
	t.Error("start mesoscopic")
}
