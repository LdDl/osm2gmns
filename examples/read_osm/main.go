package main

import (
	"fmt"

	"github.com/LdDl/go-gmns/generators"
	"github.com/LdDl/go-gmns/gmns/types"
	"github.com/LdDl/osm2gmns"
	"github.com/LdDl/osm2gmns/osmmacro"
	"github.com/LdDl/osm2gmns/osmmeso"
	"github.com/LdDl/osm2gmns/osmmicro"
	"github.com/LdDl/osm2gmns/osmmovement"
)

func main() {
	// Explicitly set global variables for logging (those are defaults actually)
	osm2gmns.VERBOSE = true
	osm2gmns.SUPPRESS_WARNINGS = false

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
	fmt.Println("macro nodes num", len(macroNet.Nodes))
	fmt.Println("macro links num", len(macroNet.Links))

	movements, err := generators.GenerateMovements(macroNet)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("movements_num", len(movements))

	mesoNet, err := generators.GenerateMesoscopic(macroNet, movements)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("meso nodes num", len(mesoNet.Nodes))
	fmt.Println("meso links num", len(mesoNet.Links))

	microNet, err := generators.GenerateMicroscopic(macroNet, mesoNet, movements)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("micro nodes num", len(microNet.Nodes))
	fmt.Println("micro links num", len(microNet.Links))

	// Export to CSV
	err = osmmacro.ExportToCSV(macroNet, "./output.csv")
	if err != nil {
		fmt.Println(err)
		return
	}
	err = osmmovement.ExportToCSV(movements, "./output.csv")
	if err != nil {
		fmt.Println(err)
		return
	}
	err = osmmeso.ExportToCSV(mesoNet, "./output.csv")
	if err != nil {
		fmt.Println(err)
		return
	}
	err = osmmicro.ExportToCSV(microNet, "./output.csv")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Export done")
}
