package osmmacro

import (
	"fmt"
	"sort"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/go-gmns/macro"
	"github.com/LdDl/go-gmns/utils/geomath"
	"github.com/LdDl/osm2gmns/wrappers"
	"github.com/paulmach/osm"
	"github.com/rs/zerolog/log"
)

type nodeElevation struct {
	meters *float64
	err    error
}

func NewNodeFrom(id gmns.NodeID, node *wrappers.NodeOSM) *macro.Node {
	elevation := readNodeElevation(node)
	if elevation.err != nil {
		warnNodeElevation(node.ID, elevation.err)
	}
	return newNodeFromElevation(id, node, elevation.meters)
}

func newNodeFromElevation(id gmns.NodeID, node *wrappers.NodeOSM, altitude *float64) *macro.Node {
	result := macro.NewNodeFrom(
		id,
		macro.WithNodeName(node.Name),
		macro.WithOSMHighwayTag(node.OsmData.Highway),
		macro.WithOSMNodeID(node.ID),
		macro.WithNodeControlType(node.ControlType),
		macro.WithPointGeom(node.InnerNode.Point()),
		macro.WithPointGeomEuclidean(geomath.PointToEuclidean(node.InnerNode.Point())),
	)
	if altitude != nil {
		macro.WithAltitudeMeters(*altitude)(result)
	}
	return result
}

func readNodeElevation(node *wrappers.NodeOSM) nodeElevation {
	altitude, present, err := wrappers.NodeElevation(node.InnerNode.Tags)
	if err != nil {
		return nodeElevation{err: err}
	}
	if present {
		return nodeElevation{meters: &altitude}
	}
	return nodeElevation{}
}

// Parse each used source point once, including points that do not become GMNS nodes.
func readNodeElevations(ways []*wrappers.WayOSM, nodes map[osm.NodeID]*wrappers.NodeOSM) (map[osm.NodeID]nodeElevation, error) {
	elevations := make(map[osm.NodeID]nodeElevation)
	for _, way := range ways {
		if way.IsPureCycle {
			continue
		}
		for _, id := range way.Nodes {
			_, seen := elevations[id]
			if seen {
				continue
			}
			node := nodes[id]
			if node == nil {
				return nil, fmt.Errorf("way %v: missing source node %v", way.ID, id)
			}
			elevations[id] = readNodeElevation(node)
		}
	}
	return elevations, nil
}

func warnNodeElevations(elevations map[osm.NodeID]nodeElevation) {
	ids := make([]osm.NodeID, 0)
	for id, elevation := range elevations {
		if elevation.err != nil {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(first, second int) bool {
		return ids[first] < ids[second]
	})
	for _, id := range ids {
		warnNodeElevation(id, elevations[id].err)
	}
}

func warnNodeElevation(id osm.NodeID, err error) {
	log.Warn().Str("scope", "gen_macro").Str("code", "unsupported_elevation").Int64("osm_node_id", int64(id)).Err(err).Msg("Node altitude is absent")
}
