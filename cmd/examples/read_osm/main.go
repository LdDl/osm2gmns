package main

import (
	"fmt"
	"os"

	"github.com/LdDl/go-gmns/generators"
	"github.com/LdDl/go-gmns/gmns/types"
	"github.com/LdDl/osm2gmns"
	"github.com/LdDl/osm2gmns/csvexport"
	"github.com/LdDl/osm2gmns/osmmacro"
	"github.com/LdDl/osm2gmns/osmmeso"
	"github.com/LdDl/osm2gmns/osmmicro"
	"github.com/LdDl/osm2gmns/preparation"
)

func main() {
	err := run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
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
		return err
	}
	macroNet, evidence, err := osm2gmns.GenerateMacroscopicWithEvidence(osmData, false)
	if err != nil {
		return err
	}
	fmt.Println("macro nodes num", len(macroNet.Nodes))
	fmt.Println("macro links num", len(macroNet.Links))

	movements, err := generators.GenerateMovements(macroNet)
	if err != nil {
		return err
	}
	fmt.Println("movements_num", len(movements))

	mesoNet, err := generators.GenerateMesoscopic(macroNet, movements)
	if err != nil {
		return err
	}

	fmt.Println("meso nodes num", len(mesoNet.Nodes))
	fmt.Println("meso links num", len(mesoNet.Links))

	microNet, err := generators.GenerateMicroscopic(macroNet, mesoNet, movements)
	if err != nil {
		return err
	}
	fmt.Println("micro nodes num", len(microNet.Nodes))
	fmt.Println("micro links num", len(microNet.Links))

	// Finish generation before preparing standard use definitions and movements.
	originalCodes, err := preparation.StoreGeneratedMovements(macroNet, movements)
	if err != nil {
		return err
	}
	options := preparation.DefaultUseOptions()
	err = preparation.Macro(macroNet, options)
	if err != nil {
		return err
	}
	err = preparation.Meso(mesoNet, options)
	if err != nil {
		return err
	}
	err = preparation.Micro(microNet, options)
	if err != nil {
		return err
	}
	macroDataset, err := osmmacro.BuildDataset(macroNet, osmmacro.GEOMETRY_SHARED)
	if err != nil {
		return err
	}
	err = evidence.Attach(macroNet, &macroDataset)
	if err != nil {
		return err
	}
	macroDataset.Provenance["generated_movement_codes"] = originalCodes
	mesoDataset, err := osmmeso.BuildDataset(mesoNet)
	if err != nil {
		return err
	}
	microDataset, err := osmmicro.BuildDataset(microNet)
	if err != nil {
		return err
	}
	datasets := []csvexport.Dataset{
		macroDataset,
		mesoDataset,
		microDataset,
	}
	err = csvexport.WriteDatasets("./output", datasets)
	if err != nil {
		return err
	}
	fmt.Println("Export done")
	return nil
}
