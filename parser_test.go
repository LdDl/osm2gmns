package osm2gmns

import (
	"testing"
	"time"

	"github.com/LdDl/go-gmns/generators"
	"github.com/LdDl/go-gmns/gmns/types"
	"github.com/LdDl/osm2gmns/osmmacro"
	"github.com/LdDl/osm2gmns/osmmeso"
	"github.com/LdDl/osm2gmns/osmmicro"
	"github.com/LdDl/osm2gmns/osmmovement"
	"github.com/rs/zerolog/log"
)

func TestParser(t *testing.T) {
	parser := NewParser(
		"./sample.osm",
		// "/home/dimitrii/Downloads/База данных/MoscowOSM/moscow.osm",
		// "/home/dimitrii/Downloads/База данных/tula_sovetskaya.osm"
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

	st := time.Now()
	movements, err := generators.GenerateMovements(macroNet)
	if err != nil {
		t.Error(err)
		return
	}
	if VERBOSE {
		log.Info().Str("scope", "gen_movement").Int("movements_num", len(movements)).Float64("elapsed", time.Since(st).Seconds()).Msg("Generating movements done!")
	}

	err = osmmacro.ExportToCSV(macroNet, "test_data/osm-NEW_test.csv")
	if err != nil {
		t.Error(err)
		return
	}
	err = osmmovement.ExportToCSV(movements, "test_data/osm-NEW_test_movement.csv")
	if err != nil {
		t.Error(err)
		return
	}

	generators.VERBOSE = VERBOSE
	mesoNet, err := generators.GenerateMesoscopic(macroNet, movements)
	if err != nil {
		t.Error(err)
		return
	}
	if VERBOSE {
		log.Info().Str("scope", "gen_meso").Int("meso_nodes_num", len(mesoNet.Nodes)).Int("meso_links_num", len(mesoNet.Links)).Float64("elapsed", time.Since(st).Seconds()).Msg("Generating meso done!")
	}

	err = osmmeso.ExportToCSV(mesoNet, "test_data/osm-NEW_test_meso.csv")
	if err != nil {
		t.Error(err)
		return
	}

	st = time.Now()
	microNet, err := generators.GenerateMicroscopic(macroNet, mesoNet, movements)
	if err != nil {
		t.Error(err)
		return
	}
	if VERBOSE {
		log.Info().Str("scope", "gen_micro").Int("micro_nodes_num", len(microNet.Nodes)).Int("micro_links_num", len(microNet.Links)).Float64("elapsed", time.Since(st).Seconds()).Msg("Generating micro done!")
	}

	err = osmmicro.ExportToCSV(microNet, "test_data/osm-NEW_test_micro.csv")
	if err != nil {
		t.Error(err)
		return
	}
}
