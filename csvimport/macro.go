// Package csvimport reads the supported standard macro tables for meso preparation.
package csvimport

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/go-gmns/macro"
)

type descriptor struct {
	Resources []resource
}

type resource struct {
	Name     string
	Path     string
	Format   string
	Encoding string
	Dialect  dialect
}

type dialect struct {
	Delimiter string
	Header    *bool
}

type table struct {
	header []string
	rows   [][]string
}

// ReadMacro reads local resources declared by a datapackage, independent of filenames.
// Unsupported tables and columns are errors, never silently discarded.
// Provenance is not used to reconstruct standard records or generator caches.
func ReadMacro(files fs.FS, descriptorPath string) (*macro.Net, error) {
	data, err := fs.ReadFile(files, descriptorPath)
	if err != nil {
		return nil, err
	}
	var description descriptor
	err = json.Unmarshal(data, &description)
	if err != nil {
		return nil, fmt.Errorf("datapackage: %w", err)
	}
	tables := make(map[string]*table)
	for _, resource := range description.Resources {
		_, exists := tables[resource.Name]
		if exists {
			return nil, fmt.Errorf("duplicate resource %q", resource.Name)
		}
		columns, supported := supportedColumns[resource.Name]
		if !supported {
			return nil, fmt.Errorf("resource %q: baseline macro import does not support this table", resource.Name)
		}
		if !fs.ValidPath(resource.Path) || strings.Contains(resource.Path, "\\") {
			return nil, fmt.Errorf("resource %q: expected a local relative CSV path", resource.Name)
		}
		if resource.Format != "" && resource.Format != "csv" {
			return nil, fmt.Errorf("resource %q: only CSV is supported", resource.Name)
		}
		if resource.Encoding != "" && !strings.EqualFold(resource.Encoding, "utf-8") {
			return nil, fmt.Errorf("resource %q: only UTF-8 is supported", resource.Name)
		}
		if resource.Dialect.Header != nil && !*resource.Dialect.Header {
			return nil, fmt.Errorf("resource %q: CSV headers are required", resource.Name)
		}
		filename := path.Join(path.Dir(descriptorPath), resource.Path)
		tables[resource.Name], err = readTable(files, filename, resource.Dialect.Delimiter, columns)
		if err != nil {
			return nil, fmt.Errorf("resource %q: %w", resource.Name, err)
		}
	}
	for _, name := range []string{"config", "node", "link"} {
		if tables[name] == nil {
			return nil, fmt.Errorf("missing required resource %q", name)
		}
	}
	if len(tables["config"].rows) != 1 {
		return nil, fmt.Errorf("config requires exactly one record")
	}
	net := macro.NewNet()
	for _, name := range []string{"config", "geometry", "zone", "use_definition", "use_group", "node", "link", "lane", "segment", "segment_lane", "movement"} {
		table := tables[name]
		if table == nil {
			continue
		}
		for index, cells := range table.rows {
			row := csvRow{values: make(map[string]string)}
			for column, name := range table.header {
				row.values[name] = cells[column]
			}
			err = readRecord(net, name, &row)
			if err != nil {
				return nil, fmt.Errorf("%s row %d: %w", name, index+2, err)
			}
		}
	}
	err = net.ValidateDataset()
	if err != nil {
		return nil, err
	}
	// Resolve shared shapes on newly read records; no external cache is required.
	linkIDs := make([]gmns.LinkID, 0, len(net.Links))
	for id := range net.Links {
		linkIDs = append(linkIDs, id)
	}
	sort.Slice(linkIDs, func(first, second int) bool {
		return linkIDs[first].Less(linkIDs[second])
	})
	for _, id := range linkIDs {
		link := net.Links[id]
		geometryID, shared := link.GeometryID()
		if !shared || len(link.Geom()) != 0 {
			continue
		}
		direction, present := link.DirFlag()
		if !present {
			return nil, fmt.Errorf("link %v dir_flag: supply the shared geometry orientation", link.ID)
		}
		geometry, _ := net.Geometry(geometryID)
		shape := geometry.Geom()
		if direction == gmns.DIR_FLAG_REVERSE {
			shape.Reverse()
		}
		macro.WithLineGeom(shape)(link)
	}
	return net, nil
}

func readTable(files fs.FS, filename, delimiter, columns string) (*table, error) {
	file, err := files.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	if delimiter != "" {
		if !utf8.ValidString(delimiter) || utf8.RuneCountInString(delimiter) != 1 {
			return nil, fmt.Errorf("delimiter must be one UTF-8 character")
		}
		reader.Comma, _ = utf8.DecodeRuneInString(delimiter)
	}
	header, err := reader.Read()
	if err != nil {
		return nil, err
	}
	allowed := strings.Fields(columns)
	seen := make(map[string]bool)
	for _, name := range header {
		if seen[name] {
			return nil, fmt.Errorf("duplicate column %q", name)
		}
		seen[name] = true
		found := false
		for _, candidate := range allowed {
			if name == candidate {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("unsupported column %q", name)
		}
	}
	result := &table{header: header}
	for {
		cells, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, readErr
		}
		for _, cell := range cells {
			if !utf8.ValidString(cell) {
				return nil, fmt.Errorf("invalid UTF-8 cell")
			}
		}
		result.rows = append(result.rows, cells)
	}
	return result, nil
}

var supportedColumns = map[string]string{
	"lane":           "lane_id link_id lane_num allowed_uses r_barrier l_barrier width",
	"segment":        "segment_id link_id ref_node_id start_lr end_lr grade capacity free_speed lanes l_lanes_added r_lanes_added bike_facility ped_facility parking allowed_uses toll jurisdiction row_width",
	"segment_lane":   "segment_lane_id segment_id lane_num parent_lane_id allowed_uses r_barrier l_barrier width",
	"config":         "dataset_name short_length long_length speed crs geometry_field_format currency version_number id_type",
	"node":           "node_id name x_coord y_coord z_coord node_type ctrl_type zone_id parent_node_id",
	"link":           "link_id name from_node_id to_node_id directed geometry_id geometry parent_link_id dir_flag length grade facility_type capacity free_speed lanes bike_facility ped_facility parking allowed_uses toll jurisdiction row_width",
	"movement":       "mvmt_id node_id name ib_link_id start_ib_lane end_ib_lane ob_link_id start_ob_lane end_ob_lane type penalty capacity ctrl_type mvmt_code allowed_uses geometry",
	"geometry":       "geometry_id geometry",
	"zone":           "zone_id name boundary super_zone",
	"use_definition": "use persons_per_vehicle pce special_conditions description",
	"use_group":      "use_group uses description",
}
