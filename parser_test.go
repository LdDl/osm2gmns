package osm2gmns

import (
	"testing"

	"github.com/LdDl/go-gmns/generators"
	"github.com/LdDl/go-gmns/gmns/types"
	"github.com/LdDl/osm2gmns/expmacro"
	"github.com/LdDl/osm2gmns/expmovement"
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

	macroNet, err := GenerateMacroscopic(osmData, parser.preparePOI)
	if err != nil {
		t.Error(err)
		return
	}
	movements, err := generators.GenerateMovements(macroNet)
	if err != nil {
		t.Error(err)
		return
	}

	err = expmacro.ExportToCSV(macroNet, "test_data/NEW_test.csv")
	if err != nil {
		t.Error(err)
		return
	}
	err = expmovement.ExportToCSV(movements, "test_data/NEW_test_movement.csv")
	if err != nil {
		t.Error(err)
		return
	}

	mesoNet, err := GenerateMesoscopic(macroNet, movements)
	if err != nil {
		t.Error(err)
		return
	}

	// @todo
	t.Error("start export mesoscopic")
	err = mesoNet.ExportToCSV("test_data/NEW_test_meso.csv")
	if err != nil {
		t.Error(err)
		return
	}
}
