package main

import (
	"fmt"
	"time"

	"github.com/LdDl/go-gmns/generators"
	"github.com/LdDl/go-gmns/gmns/types"
	"github.com/LdDl/osm2gmns"
	"github.com/rs/zerolog/log"
)

func main() {
	parser := osm2gmns.NewParser(
		"./sample.osm",
		osm2gmns.WithPreparePOI(false),
		osm2gmns.WithStrictMode(false),
		osm2gmns.WithVerbose(true),
		osm2gmns.WithAllowedAgentTypes(types.AGENT_TYPES_DEFAULT),
	)
	fmt.Println(parser)

	osmData, err := parser.ReadOSM()
	if err != nil {
		fmt.Println(err)
		return
	}
	macroNet, err := osm2gmns.GenerateMacroscopic(osmData, false)
	if err != nil {
		fmt.Println(err)
		return
	}

	st := time.Now()
	movements, err := generators.GenerateMovements(macroNet)
	if err != nil {
		fmt.Println(err)
		return
	}
	log.Info().Str("scope", "gen_movement").Int("movements_num", len(movements)).Float64("elapsed", time.Since(st).Seconds()).Msg("Generating movements done!")

	mesoNet, err := generators.GenerateMesoscopic(macroNet, movements)
	if err != nil {
		fmt.Println(err)
		return
	}
	log.Info().Str("scope", "gen_meso").Int("meso_nodes_num", len(mesoNet.Nodes)).Int("meso_links_num", len(mesoNet.Links)).Float64("elapsed", time.Since(st).Seconds()).Msg("Generating meso done!")
}
