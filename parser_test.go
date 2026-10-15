package osm2gmns

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/LdDl/go-gmns/generators"
	"github.com/LdDl/go-gmns/gmns/types"
	"github.com/LdDl/osm2gmns/osmmacro"
	"github.com/LdDl/osm2gmns/osmmeso"
	"github.com/LdDl/osm2gmns/osmmicro"
	"github.com/LdDl/osm2gmns/osmmovement"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func TestParser(t *testing.T) {
	outputDir := t.TempDir()
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

	assert.NoError(t, macroNet.ValidateDataset())
	st := time.Now()
	movements, err := generators.GenerateMovements(macroNet)
	if err != nil {
		t.Error(err)
		return
	}
	if VERBOSE {
		log.Info().Str("scope", "gen_movement").Int("movements_num", len(movements)).Float64("elapsed", time.Since(st).Seconds()).Msg("Generating movements done!")
	}

	for _, mvmt := range movements {
		assert.NoError(t, macroNet.AddMovement(mvmt))
	}
	assert.NoError(t, macroNet.ValidateDataset())
	err = osmmacro.ExportToCSV(macroNet, outputDir)
	if err != nil {
		t.Error(err)
		return
	}
	err = osmmovement.ExportToCSV(movements, filepath.Join(outputDir, "movement.csv"))
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

	assert.NoError(t, mesoNet.ValidateDataset())
	err = osmmeso.ExportToCSV(mesoNet, outputDir)
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

	assert.NoError(t, microNet.ValidateDataset())
	err = osmmicro.ExportToCSV(microNet, outputDir)
	if err != nil {
		t.Error(err)
		return
	}
}
