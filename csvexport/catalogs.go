package csvexport

import (
	"strconv"
	"strings"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/osm2gmns/csvvalue"
)

func Config(record *gmns.Config) Table {
	table := Table{
		Name: "config",
		Header: []string{
			"dataset_name",
			"short_length",
			"long_length",
			"speed",
			"crs",
			"geometry_field_format",
			"currency",
			"version_number",
			"id_type",
		},
	}
	table.Rows = append(table.Rows, []string{
		record.DatasetName(),
		record.ShortLength(),
		record.LongLength(),
		record.Speed(),
		record.CRS(),
		"WKT",
		record.Currency(),
		csvvalue.Float(record.VersionNumber()),
		string(record.IDType()),
	})
	return table
}

func Geometries(records []*gmns.Geometry) Table {
	table := Table{
		Name: "geometry",
		Header: []string{
			"geometry_id",
			"geometry",
		},
	}
	for _, record := range records {
		table.Rows = append(table.Rows, []string{
			record.ID.String(),
			Line(record.Geom()),
		})
	}
	return table
}

func Zones(records []*gmns.Zone) Table {
	table := Table{
		Name: "zone",
		Header: []string{
			"zone_id",
			"name",
			"boundary",
			"super_zone",
		},
	}
	for _, record := range records {
		table.Rows = append(table.Rows, []string{
			record.ID.String(),
			record.Name(),
			Boundary(record.Boundary()),
			csvvalue.NodeID(record.SuperZone()),
		})
	}
	return table
}

func UseDefinitions(records []*gmns.UseDefinition) Table {
	table := Table{
		Name: "use_definition",
		Header: []string{
			"use",
			"persons_per_vehicle",
			"pce",
			"special_conditions",
			"description",
		},
	}
	for _, record := range records {
		table.Rows = append(table.Rows, []string{
			record.Use(),
			Number(record.PersonsPerVehicle()),
			Number(record.PCE()),
			record.SpecialConditions(),
			record.Description(),
		})
	}
	return table
}

func UseGroups(records []*gmns.UseGroup) Table {
	table := Table{
		Name: "use_group",
		Header: []string{
			"use_group",
			"uses",
			"description",
		},
	}
	for _, record := range records {
		table.Rows = append(table.Rows, []string{
			record.Name(),
			strings.Join(record.Uses(), ","),
			record.Description(),
		})
	}
	return table
}

func Lanes(records []*gmns.Lane) Table {
	table := Table{
		Name: "lane",
		Header: []string{
			"lane_id",
			"link_id",
			"lane_num",
			"allowed_uses",
			"r_barrier",
			"l_barrier",
			"width",
		},
	}
	for _, record := range records {
		table.Rows = append(table.Rows, []string{
			record.ID.String(),
			record.Link().String(),
			strconv.Itoa(record.LaneNum()),
			record.AllowedUses(),
			record.RightBarrier(),
			record.LeftBarrier(),
			csvvalue.Float(record.WidthMeters()),
		})
	}
	return table
}

func Segments(records []*gmns.Segment) Table {
	table := Table{
		Name: "segment",
		Header: []string{
			"segment_id",
			"link_id",
			"ref_node_id",
			"start_lr",
			"end_lr",
			"grade",
			"capacity",
			"free_speed",
			"lanes",
			"l_lanes_added",
			"r_lanes_added",
			"bike_facility",
			"ped_facility",
			"parking",
			"allowed_uses",
			"toll",
			"jurisdiction",
			"row_width",
		},
	}
	for _, record := range records {
		table.Rows = append(table.Rows, []string{
			record.ID.String(),
			record.Link().String(),
			record.RefNode().String(),
			Number(record.StartMeters()),
			Number(record.EndMeters()),
			csvvalue.Float(record.Grade()),
			csvvalue.Float(record.Capacity()),
			csvvalue.Float(record.FreeSpeed()),
			csvvalue.Int(record.LanesNum()),
			csvvalue.Int(record.LeftLanesAdded()),
			csvvalue.Int(record.RightLanesAdded()),
			record.BikeFacility(),
			record.PedFacility(),
			record.Parking(),
			record.AllowedUses(),
			csvvalue.Float(record.Toll()),
			record.Jurisdiction(),
			csvvalue.Float(record.RightOfWayWidthMeters()),
		})
	}
	return table
}

func SegmentLanes(records []*gmns.SegmentLane) Table {
	table := Table{
		Name: "segment_lane",
		Header: []string{
			"segment_lane_id",
			"segment_id",
			"lane_num",
			"parent_lane_id",
			"allowed_uses",
			"r_barrier",
			"l_barrier",
			"width",
		},
	}
	for _, record := range records {
		table.Rows = append(table.Rows, []string{
			record.ID.String(),
			record.Segment().String(),
			strconv.Itoa(record.LaneNum()),
			LaneID(record.ParentLane()),
			record.AllowedUses(),
			record.RightBarrier(),
			record.LeftBarrier(),
			csvvalue.Float(record.WidthMeters()),
		})
	}
	return table
}
