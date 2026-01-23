package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/LdDl/go-gmns/generators"
	"github.com/LdDl/go-gmns/gmns/types"
	"github.com/LdDl/go-gmns/meso"
	"github.com/LdDl/go-gmns/movement"
	"github.com/LdDl/osm2gmns"
	"github.com/LdDl/osm2gmns/osmmacro"
	"github.com/LdDl/osm2gmns/osmmeso"
	"github.com/LdDl/osm2gmns/osmmicro"
	"github.com/LdDl/osm2gmns/osmmovement"
)

var (
	version   = "dev"
	buildTime = "unknown"
	gitCommit = "unknown"
)

func main() {
	// Define flags
	inputFile := flag.String("input", "", "Input OSM file path (.osm or .osm.pbf)")
	outputDir := flag.String("output", "./output", "Output directory for CSV files")
	networkLevels := flag.String("networks", "macro,movement,meso,micro", "Comma-separated network levels to generate: macro,movement,meso,micro")
	agentTypes := flag.String("agents", "auto,bike,walk", "Comma-separated allowed agent types: auto,bike,walk")
	verbose := flag.Bool("verbose", true, "Enable verbose output")
	suppressWarnings := flag.Bool("suppress-warnings", false, "Suppress warning messages")
	strictMode := flag.Bool("strict", false, "Enable strict mode (fail on warnings)")
	preparePOI := flag.Bool("poi", false, "Prepare POI data")
	showVersion := flag.Bool("version", false, "Show version information")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "osm2gmns - Convert OpenStreetMap data to GMNS format\n\n")
		fmt.Fprintf(os.Stderr, "Usage: osm2gmns [options]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  osm2gmns -input map.osm -output ./results\n")
		fmt.Fprintf(os.Stderr, "  osm2gmns -input map.osm.pbf -networks macro,meso -agents auto\n")
		fmt.Fprintf(os.Stderr, "  osm2gmns -input map.osm -output ./data -verbose=false\n")
	}

	flag.Parse()

	if *showVersion {
		fmt.Printf("osm2gmns %s\n", version)
		fmt.Printf("  Build time: %s\n", buildTime)
		fmt.Printf("  Git commit: %s\n", gitCommit)
		os.Exit(0)
	}

	if *inputFile == "" {
		fmt.Fprintln(os.Stderr, "Error: input file is required")
		flag.Usage()
		os.Exit(1)
	}

	// Check input file exists
	if _, err := os.Stat(*inputFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: input file does not exist: %s\n", *inputFile)
		os.Exit(1)
	}

	// Parse network levels
	levels := parseNetworkLevels(*networkLevels)
	if len(levels) == 0 {
		fmt.Fprintln(os.Stderr, "Error: at least one network level must be specified")
		os.Exit(1)
	}

	// Validate network level dependencies
	if err := validateLevelDependencies(levels); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Parse agent types
	agents := parseAgentTypes(*agentTypes)

	// Create output directory
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create output directory: %v\n", err)
		os.Exit(1)
	}

	// Set global logging options
	osm2gmns.VERBOSE = *verbose
	osm2gmns.SUPPRESS_WARNINGS = *suppressWarnings

	if *verbose {
		fmt.Printf("osm2gmns %s\n", version)
		fmt.Printf("Input file: %s\n", *inputFile)
		fmt.Printf("Output directory: %s\n", *outputDir)
		fmt.Printf("Network levels: %v\n", levels)
		fmt.Printf("Agent types: %v\n", agents)
		fmt.Println()
	}

	// Create parser
	parser := osm2gmns.NewParser(
		*inputFile,
		osm2gmns.WithPreparePOI(*preparePOI),
		osm2gmns.WithStrictMode(*strictMode),
		osm2gmns.WithVerbose(*verbose),
		osm2gmns.WithAllowedAgentTypes(agents),
	)

	// Read OSM data
	if *verbose {
		fmt.Println("Reading OSM data...")
	}
	osmData, err := parser.ReadOSM()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading OSM file: %v\n", err)
		os.Exit(1)
	}

	// Generate macro network
	if *verbose {
		fmt.Println("Generating macroscopic network...")
	}
	macroNet, err := osm2gmns.GenerateMacroscopic(osmData, *preparePOI)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating macro network: %v\n", err)
		os.Exit(1)
	}
	if *verbose {
		fmt.Printf("  Macro nodes: %d\n", len(macroNet.Nodes))
		fmt.Printf("  Macro links: %d\n", len(macroNet.Links))
	}

	// Export macro network
	if levels["macro"] {
		if err := osmmacro.ExportToCSV(macroNet, *outputDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error exporting macro network: %v\n", err)
			os.Exit(1)
		}
		if *verbose {
			fmt.Printf("  Exported: %s, %s\n",
				filepath.Join(*outputDir, "node.csv"),
				filepath.Join(*outputDir, "link.csv"))
		}
	}

	// Generate movements if needed
	var movements movement.MovementsStorage
	if levels["movement"] || levels["meso"] || levels["micro"] {
		if *verbose {
			fmt.Println("Generating movements...")
		}
		movements, err = generators.GenerateMovements(macroNet)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating movements: %v\n", err)
			os.Exit(1)
		}
		if *verbose {
			fmt.Printf("  Movements: %d\n", len(movements))
		}

		if levels["movement"] {
			if err := osmmovement.ExportToCSV(movements, filepath.Join(*outputDir, "movement.csv")); err != nil {
				fmt.Fprintf(os.Stderr, "Error exporting movements: %v\n", err)
				os.Exit(1)
			}
			if *verbose {
				fmt.Printf("  Exported: %s\n", filepath.Join(*outputDir, "movement.csv"))
			}
		}
	}

	// Generate meso network if needed
	var mesoNet *meso.Net
	if levels["meso"] || levels["micro"] {
		if *verbose {
			fmt.Println("Generating mesoscopic network...")
		}
		mesoNet, err = generators.GenerateMesoscopic(macroNet, movements)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating meso network: %v\n", err)
			os.Exit(1)
		}
		if *verbose {
			fmt.Printf("  Meso nodes: %d\n", len(mesoNet.Nodes))
			fmt.Printf("  Meso links: %d\n", len(mesoNet.Links))
		}

		if levels["meso"] {
			if err := osmmeso.ExportToCSV(mesoNet, *outputDir); err != nil {
				fmt.Fprintf(os.Stderr, "Error exporting meso network: %v\n", err)
				os.Exit(1)
			}
			if *verbose {
				fmt.Printf("  Exported: %s, %s\n",
					filepath.Join(*outputDir, "mesonode.csv"),
					filepath.Join(*outputDir, "mesolink.csv"))
			}
		}
	}

	// Generate micro network if needed
	if levels["micro"] {
		if *verbose {
			fmt.Println("Generating microscopic network...")
		}
		microNet, err := generators.GenerateMicroscopic(macroNet, mesoNet, movements)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating micro network: %v\n", err)
			os.Exit(1)
		}
		if *verbose {
			fmt.Printf("  Micro nodes: %d\n", len(microNet.Nodes))
			fmt.Printf("  Micro links: %d\n", len(microNet.Links))
		}

		if err := osmmicro.ExportToCSV(microNet, *outputDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error exporting micro network: %v\n", err)
			os.Exit(1)
		}
		if *verbose {
			fmt.Printf("  Exported: %s, %s\n",
				filepath.Join(*outputDir, "micronode.csv"),
				filepath.Join(*outputDir, "microlink.csv"))
		}
	}

	if *verbose {
		fmt.Println()
		fmt.Println("Done!")
	}
}

func parseNetworkLevels(s string) map[string]bool {
	levels := make(map[string]bool)
	for _, level := range strings.Split(s, ",") {
		level = strings.TrimSpace(strings.ToLower(level))
		switch level {
		case "macro", "movement", "meso", "micro":
			levels[level] = true
		}
	}
	return levels
}

func validateLevelDependencies(levels map[string]bool) error {
	// micro requires meso
	if levels["micro"] && !levels["meso"] {
		// Auto-enable meso for micro generation (but don't export)
		// This is handled in the main logic
	}
	// meso requires movement
	if levels["meso"] && !levels["movement"] {
		// Auto-enable movement generation (but don't export)
	}
	return nil
}

func parseAgentTypes(s string) []types.AgentType {
	var agents []types.AgentType
	for _, agent := range strings.Split(s, ",") {
		agent = strings.TrimSpace(strings.ToLower(agent))
		switch agent {
		case "auto":
			agents = append(agents, types.AGENT_AUTO)
		case "bike":
			agents = append(agents, types.AGENT_BIKE)
		case "walk":
			agents = append(agents, types.AGENT_WALK)
		}
	}
	if len(agents) == 0 {
		return types.AGENT_TYPES_DEFAULT
	}
	return agents
}
