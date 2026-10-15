package csvexport

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const schemaBaseURL = "https://raw.githubusercontent.com/zephyr-data-specs/GMNS/v0.97/spec/"

var tableNames = []string{
	"config", "node", "link", "geometry", "movement", "zone", "use_definition", "use_group", "lane", "segment", "segment_lane",
}

type fileChange struct {
	name      string
	backedUp  bool
	installed bool
}

// WriteDatasets stages all selected datasets before replacing their managed files.
// Unrelated files and unselected levels are left in place.
func WriteDatasets(outputDir string, datasets []Dataset) error {
	err := validateTables(datasets)
	if err != nil {
		return err
	}
	err = os.MkdirAll(outputDir, 0755)
	if err != nil {
		return err
	}
	staging, err := os.MkdirTemp(outputDir, ".gmns-")
	if err != nil {
		return err
	}
	keepStaging := false
	defer func() {
		if !keepStaging {
			os.RemoveAll(staging)
		}
	}()
	backup := filepath.Join(staging, "backup")
	err = os.Mkdir(backup, 0755)
	if err != nil {
		return err
	}
	files := make(map[string]bool)
	managed := make([]string, 0)
	for _, dataset := range datasets {
		resources := make([]map[string]any, 0)
		for _, table := range dataset.Tables {
			name := table.Name + "_" + dataset.Level + ".csv"
			managed = append(managed, name)
			if len(table.Rows) == 0 && table.Name != "node" && table.Name != "link" {
				continue
			}
			err = WriteTable(filepath.Join(staging, name), table)
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
			files[name] = true
			resources = append(resources, map[string]any{
				"name":     table.Name,
				"path":     name,
				"profile":  "tabular-data-resource",
				"format":   "csv",
				"encoding": "utf-8",
				"dialect":  map[string]any{"delimiter": ",", "header": true},
				"schema":   schemaBaseURL + table.Name + ".schema.json",
			})
		}
		// Remove optional tables left by a previous export of this same level.
		for _, tableName := range tableNames {
			name := tableName + "_" + dataset.Level + ".csv"
			if !containsName(managed, name) {
				managed = append(managed, name)
			}
		}
		descriptorName := "datapackage_" + dataset.Level + ".json"
		descriptor := map[string]any{
			"profile":   "tabular-data-package",
			"name":      "gmns-" + dataset.Level,
			"resources": resources,
		}
		err = writeJSON(filepath.Join(staging, descriptorName), descriptor)
		if err != nil {
			return err
		}
		managed = append(managed, descriptorName)
		files[descriptorName] = true
		metadataName := "provenance_" + dataset.Level + ".json"
		managed = append(managed, metadataName)
		if dataset.Provenance != nil {
			err = writeJSON(filepath.Join(staging, metadataName), dataset.Provenance)
			if err != nil {
				return err
			}
			files[metadataName] = true
		}
	}
	sort.Strings(managed)
	for _, name := range managed {
		info, statErr := os.Lstat(filepath.Join(outputDir, name))
		if statErr != nil && !os.IsNotExist(statErr) {
			return statErr
		}
		if statErr == nil && !info.Mode().IsRegular() {
			return fmt.Errorf("refusing to replace non-regular output %s", name)
		}
	}
	changes := make([]fileChange, 0, len(managed))
	for _, name := range managed {
		change := fileChange{name: name}
		destination := filepath.Join(outputDir, name)
		err = os.Rename(destination, filepath.Join(backup, name))
		if err == nil {
			change.backedUp = true
		} else if !os.IsNotExist(err) {
			rollbackErr := rollback(outputDir, backup, changes)
			if rollbackErr != nil {
				keepStaging = true
				return errors.Join(err, fmt.Errorf("rollback failed; backups retained in %s: %w", backup, rollbackErr))
			}
			return err
		}
		if files[name] {
			err = os.Rename(filepath.Join(staging, name), destination)
			if err != nil {
				changes = append(changes, change)
				rollbackErr := rollback(outputDir, backup, changes)
				if rollbackErr != nil {
					keepStaging = true
					return errors.Join(err, fmt.Errorf("rollback failed; backups retained in %s: %w", backup, rollbackErr))
				}
				return err
			}
			change.installed = true
		}
		changes = append(changes, change)
	}
	return nil
}

func rollback(outputDir, backup string, changes []fileChange) error {
	var result error
	for index := len(changes) - 1; index >= 0; index-- {
		change := changes[index]
		target := filepath.Join(outputDir, change.name)
		if change.installed {
			err := os.Remove(target)
			result = errors.Join(result, err)
		}
		if change.backedUp {
			err := os.Rename(filepath.Join(backup, change.name), target)
			result = errors.Join(result, err)
		}
	}
	return result
}

func writeJSON(filename string, value map[string]any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filename, data, 0644)
}

func containsName(names []string, name string) bool {
	for _, candidate := range names {
		if candidate == name {
			return true
		}
	}
	return false
}

func validateTables(datasets []Dataset) error {
	levels := make(map[string]bool)
	for _, dataset := range datasets {
		if dataset.Level != "macro" && dataset.Level != "meso" && dataset.Level != "micro" {
			return fmt.Errorf("invalid dataset level %q", dataset.Level)
		}
		if levels[dataset.Level] {
			return fmt.Errorf("duplicate dataset %q", dataset.Level)
		}
		levels[dataset.Level] = true
		seen := make(map[string]bool)
		for _, table := range dataset.Tables {
			if !containsName(tableNames, table.Name) || seen[table.Name] {
				return fmt.Errorf("invalid or duplicate table %q", table.Name)
			}
			seen[table.Name] = true
			if table.Name == "config" && len(table.Rows) != 1 {
				return fmt.Errorf("config requires exactly one row")
			}
			err := validateTable(table)
			if err != nil {
				return err
			}
		}
		if !seen["config"] || !seen["node"] || !seen["link"] {
			return fmt.Errorf("dataset %s requires config, node and link tables", dataset.Level)
		}
	}
	return nil
}
