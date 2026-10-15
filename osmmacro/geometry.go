package osmmacro

import (
	"fmt"
	"math"
	"sort"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/go-gmns/macro"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/encoding/wkt"
)

type GeometryMode string

const (
	GEOMETRY_INLINE GeometryMode = "inline"
	GEOMETRY_SHARED GeometryMode = "shared"
)

// collectGeometries validates prepared canonical records without rebuilding or allocating them.
func collectGeometries(macroNet *macro.Net) (map[gmns.GeometryID]orb.LineString, error) {
	if macroNet == nil {
		return nil, fmt.Errorf("nil macro network")
	}
	geometries := make(map[gmns.GeometryID]orb.LineString)
	for _, geometry := range macroNet.GeometryRecords() {
		err := geometry.Validate()
		if err != nil {
			return nil, err
		}
		geometries[geometry.ID] = geometry.Geom()
	}
	for nodeID, node := range macroNet.Nodes {
		if node == nil || node.ID != nodeID {
			return nil, fmt.Errorf("node %v: nil node or mismatched map key", nodeID)
		}
	}
	for linkID, link := range macroNet.Links {
		if link == nil || link.ID != linkID {
			return nil, fmt.Errorf("link %v: nil link or mismatched map key", linkID)
		}
		geometryID, hasGeometryID := link.GeometryID()
		if !hasGeometryID {
			return nil, fmt.Errorf("link %v: shared geometry requires geometry_id", linkID)
		}
		dirFlag, hasDirFlag := link.DirFlag()
		if !hasDirFlag || (dirFlag != gmns.DIR_FLAG_FORWARD && dirFlag != gmns.DIR_FLAG_REVERSE) {
			return nil, fmt.Errorf("link %v: shared geometry requires dir_flag 1 or -1", linkID)
		}
		shape := link.Geom()
		if len(shape) < 2 {
			return nil, fmt.Errorf("link %v: shared geometry requires at least two points", linkID)
		}
		for _, point := range shape {
			if math.IsNaN(point[0]) || math.IsNaN(point[1]) || math.IsInf(point[0], 0) || math.IsInf(point[1], 0) {
				return nil, fmt.Errorf("link %v: geometry coordinates must be finite", linkID)
			}
		}
		source, hasSource := macroNet.Nodes[link.SourceNode()]
		target, hasTarget := macroNet.Nodes[link.TargetNode()]
		if !hasSource || !hasTarget {
			return nil, fmt.Errorf("link %v: missing source or target node", linkID)
		}
		if shape[0] != source.Geom() || shape[len(shape)-1] != target.Geom() {
			return nil, fmt.Errorf("link %v: Geom() must follow source and target node coordinates", linkID)
		}
		canonical, exists := geometries[geometryID]
		if !exists {
			return nil, fmt.Errorf("link %v: geometry_id %v has no prepared geometry record", linkID, geometryID)
		}
		if len(canonical) != len(shape) {
			return nil, fmt.Errorf("link %v: geometry_id %v refers to contradictory shapes", linkID, geometryID)
		}
		for index, point := range shape {
			canonicalIndex := index
			if dirFlag == gmns.DIR_FLAG_REVERSE {
				canonicalIndex = len(shape) - 1 - index
			}
			if point != canonical[canonicalIndex] {
				return nil, fmt.Errorf("link %v: geometry_id %v contradicts its shape or dir_flag", linkID, geometryID)
			}
		}
	}
	return geometries, nil
}

func exportGeometriesToCSV(geometries map[gmns.GeometryID]orb.LineString, filename string) error {
	geometryIDs := make([]gmns.GeometryID, 0, len(geometries))
	for geometryID := range geometries {
		geometryIDs = append(geometryIDs, geometryID)
	}
	sort.Slice(geometryIDs, func(first, second int) bool {
		return geometryIDs[first].Less(geometryIDs[second])
	})
	file, writer, err := newCSVFile(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	err = writer.Write([]string{"geometry_id", "geometry"})
	if err != nil {
		return err
	}
	for _, geometryID := range geometryIDs {
		err = writer.Write([]string{geometryID.String(), wkt.MarshalString(geometries[geometryID])})
		if err != nil {
			return err
		}
	}
	return finishCSVFile(file, writer)
}
