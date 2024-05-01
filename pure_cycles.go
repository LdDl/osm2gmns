package osm2gmns

import (
	"time"

	"github.com/paulmach/osm"
	"github.com/rs/zerolog/log"
)

// markPureCycles marks pure cycles for given set of ways and reference info about nodes
func markPureCycles(nodesSet map[osm.NodeID]*Node, ways []*WayOSM) error {
	if VERBOSE {
		log.Info().Str("scope", "ispect_pure_cycles").Int("nodes_num", len(nodesSet)).Int("ways_num", len(ways)).Msg("Marking pure cycles")
	}
	st := time.Now()
	cyclesNum := 0
	pureCyclesNum := 0
	for i := range ways {
		way := ways[i]
		// Find and mark pure cycles
		if way.isCycle {
			cyclesNum++
			// Assume that way has pure cycle
			way.isPureCycle = true
			for _, nodeID := range way.Nodes {
				existingNode, ok := nodesSet[nodeID]
				if !ok {
					log.Warn().Str("scope", "ispect_pure_cycles").Any("osm_way_id", way.ID).Int("node_id", int(nodeID)).Msg("Can't find way node in nodes set")
					return nil
				}
				if existingNode.isCrossing {
					// Way has not pure cycle if child node is cross
					way.isPureCycle = false
				}
			}
			if way.isPureCycle {
				pureCyclesNum++
			}
		}
	}
	if VERBOSE {
		log.Info().Str("scope", "ispect_pure_cycles").Int("cycles_num", cyclesNum).Int("pure_cycles_num", pureCyclesNum).Float64("elapsed", time.Since(st).Seconds()).Msg("Marking pure cycles done!")
	}
	return nil
}
