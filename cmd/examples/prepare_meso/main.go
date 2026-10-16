package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/LdDl/go-gmns/generators"
	"github.com/LdDl/osm2gmns/csvexport"
	"github.com/LdDl/osm2gmns/csvimport"
	"github.com/LdDl/osm2gmns/osmmacro"
	"github.com/LdDl/osm2gmns/osmmeso"
)

func main() {
	err := run(os.Args[1:], os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(arguments []string, output io.Writer) error {
	flags := flag.NewFlagSet("prepare_meso", flag.ContinueOnError)
	flags.SetOutput(output)
	input := flags.String("input", "", "Macro datapackage JSON path")
	destination := flags.String("output", "", "New output directory")
	layout := flags.String("lane-layout", "", "Accepted layout: uniform or detailed (uniform gaps)")
	options := make([]generators.MesoPreparationOption, 0)
	connections := flags.String("lane-connections", "single", "Movement lane policy: single or ordered (equal-sized ranges)")
	var defaultLanes *int
	flags.Func("default-lanes", "Explicit lane count for links with absent lanes only (1..99)", func(value string) error {
		count, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		if count < 1 || count > 99 {
			return fmt.Errorf("default lanes must be in 1..99")
		}
		defaultLanes = &count
		options = append(options, generators.WithDefaultMesoLanes(count))
		return nil
	})
	delimiter := flags.String("delimiter", ";", "Output CSV delimiter; input dialect comes from each resource")
	err := flags.Parse(arguments)
	if err != nil {
		return err
	}
	if *input == "" || *destination == "" || flags.NArg() != 0 {
		return fmt.Errorf("provide -input <datapackage.json> -output <new-directory> -lane-layout uniform")
	}
	switch *layout {
	case "uniform":
		options = append(options, generators.WithUniformLaneLayout())
	case "detailed":
		options = append(options, generators.WithDetailedLaneLayout())
	default:
		return fmt.Errorf("explicitly accept -lane-layout uniform or detailed")
	}
	switch *connections {
	case "single":
	case "ordered":
		options = append(options, generators.WithOrderedLaneConnections())
	default:
		return fmt.Errorf("lane-connections must be single or ordered")
	}
	separator, err := csvexport.ParseDelimiter(*delimiter)
	if err != nil {
		return err
	}
	_, err = os.Lstat(*destination)
	if err == nil {
		return fmt.Errorf("output directory already exists: %s; choose a new directory", *destination)
	}
	if !os.IsNotExist(err) {
		return err
	}
	source, err := csvimport.ReadMacro(os.DirFS(filepath.Dir(*input)), filepath.Base(*input))
	if err != nil {
		return err
	}
	prepared, err := generators.PrepareMesoscopic(source, options...)
	if err != nil {
		return err
	}
	macroDataset, err := osmmacro.BuildDataset(prepared.Macro(), osmmacro.GEOMETRY_INLINE)
	if err != nil {
		return err
	}
	mesoDataset, err := osmmeso.BuildDataset(prepared.Meso())
	if err != nil {
		return err
	}
	metadata := map[string]any{
		"algorithm":            "standard-macro-sections-3",
		"lane_layout":          *layout,
		"movement_policy":      *connections,
		"connector_policy":     "explicit",
		"allowed_uses_policy":  "intersection-of-supplied-constraints",
		"missing_lanes_policy": "reject",
	}
	if defaultLanes != nil {
		metadata["missing_lanes_policy"] = "constant"
		metadata["default_lanes"] = *defaultLanes
	}
	assumptions := make([]map[string]any, 0)
	for _, assumption := range prepared.LaneAssumptions() {
		assumptions = append(assumptions, map[string]any{
			"link_id": assumption.Link(),
			"field":   "lanes",
			"value":   assumption.Lanes(),
			"reason":  "explicit-default-for-absent-value",
		})
	}
	metadata["assumptions"] = assumptions
	metadata["sections"] = sectionMetadata(prepared)
	metadata["connections"] = connectionMetadata(prepared)
	macroDataset.Provenance["preparation"] = metadata
	mesoDataset.Provenance["preparation"] = metadata
	// Reserve a new directory only after preparation and serialization have succeeded.
	err = os.Mkdir(*destination, 0755)
	if err != nil {
		return err
	}
	datasets := []csvexport.Dataset{macroDataset, mesoDataset}
	err = csvexport.WriteDatasets(*destination, datasets, csvexport.WithDelimiter(separator))
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(output, "Prepared meso: %d nodes, %d links; macro: %d nodes, %d links, %d movements\n",
		len(prepared.Meso().Nodes), len(prepared.Meso().Links), len(source.Nodes), len(source.Links), len(source.MovementRecords()))
	return err
}

func sectionMetadata(result *generators.MesoPreparationResult) []map[string]any {
	records := make([]map[string]any, 0)
	for _, section := range result.Sections() {
		record := map[string]any{
			"meso_link_id":            section.Link(),
			"source_link_id":          section.SourceLink(),
			"source_start_meters":     section.StartMeters(),
			"source_end_meters":       section.EndMeters(),
			"lane_numbers":            section.LaneNumbers(),
			"source_lane_ids":         section.SourceLanes(),
			"source_segment_lane_ids": section.SourceSegmentLanes(),
		}
		segment, present := section.Segment()
		if present {
			record["source_segment_id"] = segment
		}
		records = append(records, record)
	}
	return records
}

func connectionMetadata(result *generators.MesoPreparationResult) []map[string]any {
	records := make([]map[string]any, 0)
	for _, connection := range result.Connections() {
		records = append(records, map[string]any{
			"meso_link_id":       connection.Link(),
			"source_movement_id": connection.Movement(),
			"inbound_lanes":      connection.InboundLanes(),
			"outbound_lanes":     connection.OutboundLanes(),
		})
	}
	return records
}
