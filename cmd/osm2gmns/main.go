package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/LdDl/go-gmns/generators"
	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/go-gmns/gmns/types"
	"github.com/LdDl/go-gmns/meso"
	"github.com/LdDl/go-gmns/micro"
	"github.com/LdDl/osm2gmns"
	"github.com/LdDl/osm2gmns/csvexport"
	"github.com/LdDl/osm2gmns/osmmacro"
	"github.com/LdDl/osm2gmns/osmmeso"
	"github.com/LdDl/osm2gmns/osmmicro"
	"github.com/LdDl/osm2gmns/preparation"
)

var (
	version   = "dev"
	buildTime = "unknown"
	gitCommit = "unknown"
)

func main() {
	err := run(os.Args[1:], os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(arguments []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("osm2gmns", flag.ContinueOnError)
	flags.SetOutput(stderr)
	inputFile := flags.String("input", "", "Input OSM file (.osm or .osm.pbf)")
	outputDir := flags.String("output", "./output", "Flat output directory for GMNS datasets")
	delimiterFlag := flags.String("delimiter", string(csvexport.DefaultDelimiter), "CSV separator: one character")
	sourceMetadata := flags.Bool("source-metadata", true, "Include optional OSM source evidence in macro provenance")
	geometryMode := flags.String("geometry", "inline", "Macro geometry: inline or shared; meso/micro remain inline")
	networkLevels := flags.String("networks", "macro,movement,meso,micro", "Levels to export: macro,meso,micro; movement also selects its macro network")
	agentTypes := flags.String("agents", "auto,bike,walk", "Allowed agents: auto,bike,walk")
	verbose := flags.Bool("verbose", true, "Enable verbose output")
	suppressWarnings := flags.Bool("suppress-warnings", false, "Suppress warning messages")
	strictMode := flags.Bool("strict", false, "Fail on warnings")
	preparePOI := flags.Bool("poi", false, "Prepare POI data")
	showVersion := flags.Bool("version", false, "Show version information")
	useOptions := preparation.DefaultUseOptions()
	flags.Float64Var(&useOptions.AutoPersonsPerVehicle, "auto-persons", preparation.DefaultAutoPersonsPerVehicle, "Modeled persons per auto vehicle")
	flags.Float64Var(&useOptions.AutoPCE, "auto-pce", preparation.DefaultAutoPCE, "Auto passenger-car equivalent")
	flags.Float64Var(&useOptions.BikePersonsPerVehicle, "bike-persons", preparation.DefaultBikePersonsPerVehicle, "Modeled persons per bicycle")
	flags.Float64Var(&useOptions.BikePCE, "bike-pce", preparation.DefaultBikePCE, "Bicycle passenger-car equivalent")
	flags.Float64Var(&useOptions.WalkPersonsPerVehicle, "walk-persons", preparation.DefaultWalkPersonsPerVehicle, "Persons per walking unit")
	flags.Float64Var(&useOptions.WalkPCE, "walk-pce", preparation.DefaultWalkPCE, "Pedestrian passenger-car equivalent")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "osm2gmns - Convert OpenStreetMap data to GMNS datasets")
		fmt.Fprintln(stderr, "Usage: osm2gmns -input map.osm -output ./results")
		flags.PrintDefaults()
	}
	if len(arguments) == 0 {
		flags.Usage()
		return nil
	}
	err := flags.Parse(arguments)
	if err == flag.ErrHelp {
		return nil
	}
	if err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %v", flags.Args())
	}
	if *showVersion {
		fmt.Fprintf(stdout, "osm2gmns %s\nBuild time: %s\nGit commit: %s\n", version, buildTime, gitCommit)
		return nil
	}
	if *inputFile == "" {
		return fmt.Errorf("input file is required")
	}
	delimiter, err := csvexport.ParseDelimiter(*delimiterFlag)
	if err != nil {
		return err
	}
	levels, err := parseNetworkLevels(*networkLevels)
	if err != nil {
		return err
	}
	agents, err := parseAgentTypes(*agentTypes)
	if err != nil {
		return err
	}
	mode := osmmacro.GeometryMode(*geometryMode)
	if mode != osmmacro.GEOMETRY_INLINE && mode != osmmacro.GEOMETRY_SHARED {
		return fmt.Errorf("unsupported geometry mode %q", mode)
	}
	if mode == osmmacro.GEOMETRY_SHARED && !levels["macro"] {
		return fmt.Errorf("-geometry shared requires macro export")
	}
	_, err = useOptions.Definitions()
	if err != nil {
		return err
	}
	osm2gmns.VERBOSE = *verbose
	osm2gmns.SUPPRESS_WARNINGS = *suppressWarnings
	generators.VERBOSE = *verbose
	parser := osm2gmns.NewParser(*inputFile,
		osm2gmns.WithPreparePOI(*preparePOI),
		osm2gmns.WithStrictMode(*strictMode),
		osm2gmns.WithVerbose(*verbose),
		osm2gmns.WithAllowedAgentTypes(agents),
	)
	source, err := parser.ReadOSM()
	if err != nil {
		return err
	}
	macroNet, evidence, err := osm2gmns.GenerateMacroscopicWithEvidence(source, *preparePOI)
	if err != nil {
		return err
	}
	if !*suppressWarnings {
		for _, diagnostic := range evidence.Diagnostics() {
			fmt.Fprintf(stderr, "OSM %s %s: %s: %s\n", diagnostic.Kind, diagnostic.ID, diagnostic.Code, diagnostic.Message)
		}
	}
	config := gmns.NewConfig(gmns.IDTypeInteger,
		gmns.WithDatasetName(filepath.Base(*inputFile)),
		gmns.WithVersionNumber(0.97),
	)
	err = macroNet.SetConfig(config)
	if err != nil {
		return err
	}
	movements, err := generators.GenerateMovements(macroNet)
	if err != nil {
		return err
	}
	var mesoNet *meso.Net
	if levels["meso"] || levels["micro"] {
		mesoNet, err = generators.GenerateMesoscopic(macroNet, movements)
		if err != nil {
			return err
		}
	}
	var microNet *micro.Net
	if levels["micro"] {
		microNet, err = generators.GenerateMicroscopic(macroNet, mesoNet, movements)
		if err != nil {
			return err
		}
	}
	// Preparation follows generation because the generators consume built-in agent permissions.
	originalCodes, err := preparation.StoreGeneratedMovements(macroNet, movements)
	if err != nil {
		return err
	}
	datasets := make([]csvexport.Dataset, 0, 3)
	if levels["macro"] {
		err = preparation.Macro(macroNet, useOptions)
		if err != nil {
			return err
		}
		dataset, buildErr := osmmacro.BuildDataset(macroNet, mode)
		if buildErr != nil {
			return buildErr
		}
		if *sourceMetadata {
			err = evidence.Attach(macroNet, &dataset)
			if err != nil {
				return err
			}
		}
		dataset.Provenance["generated_movement_codes"] = originalCodes
		datasets = append(datasets, dataset)
	}
	if levels["meso"] {
		err = preparation.Meso(mesoNet, useOptions)
		if err != nil {
			return err
		}
		dataset, buildErr := osmmeso.BuildDataset(mesoNet)
		if buildErr != nil {
			return buildErr
		}
		datasets = append(datasets, dataset)
	}
	if levels["micro"] {
		err = preparation.Micro(microNet, useOptions)
		if err != nil {
			return err
		}
		dataset, buildErr := osmmicro.BuildDataset(microNet)
		if buildErr != nil {
			return buildErr
		}
		datasets = append(datasets, dataset)
	}
	err = csvexport.WriteDatasets(*outputDir, datasets, csvexport.WithDelimiter(delimiter))
	if err != nil {
		return err
	}
	if *verbose {
		fmt.Fprintf(stdout, "Exported %d GMNS datasets to %s\n", len(datasets), *outputDir)
	}
	return nil
}

func parseNetworkLevels(value string) (map[string]bool, error) {
	levels := make(map[string]bool)
	for _, level := range strings.Split(value, ",") {
		level = strings.TrimSpace(strings.ToLower(level))
		switch level {
		case "macro", "meso", "micro":
			levels[level] = true
		case "movement":
			levels["macro"] = true
		default:
			return nil, fmt.Errorf("unknown network level %q", level)
		}
	}
	return levels, nil
}

func parseAgentTypes(value string) ([]types.AgentType, error) {
	agents := make([]types.AgentType, 0)
	seen := make(map[types.AgentType]bool)
	for _, name := range strings.Split(value, ",") {
		name = strings.TrimSpace(strings.ToLower(name))
		var agent types.AgentType
		switch name {
		case "auto":
			agent = types.AGENT_AUTO
		case "bike":
			agent = types.AGENT_BIKE
		case "walk":
			agent = types.AGENT_WALK
		default:
			return nil, fmt.Errorf("unknown agent %q", name)
		}
		if !seen[agent] {
			agents = append(agents, agent)
			seen[agent] = true
		}
	}
	return agents, nil
}
