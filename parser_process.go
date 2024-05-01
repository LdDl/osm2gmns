package osm2gmns

import (
	"time"

	"github.com/LdDl/osm2gmns/types"
	"github.com/paulmach/osm"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

func (data *OSMWaysNodes) prepareNetwork(allowedAgentTypes []types.AgentType, poi bool) error {
	preparedWays, err := prepareWays(data.ways, data.nodes, allowedAgentTypes)
	if err != nil {
		return errors.Wrap(err, "Can't prepare ways")
	}
	preparedNodes, err := prepareNodes(data.nodes)
	if err != nil {
		return errors.Wrap(err, "Can't prepare nodes")
	}
	err = markPureCycles(preparedNodes, preparedWays)
	if err != nil {
		return errors.Wrap(err, "Can't mark pure cycles")
	}
	// @todo: implement constructor for macro network
	return nil
}

// prepareNodes examines nodes which has use count > 0 and use count > 2 on being cross
func prepareNodes(nodesSet map[osm.NodeID]*Node) (map[osm.NodeID]*Node, error) {
	if VERBOSE {
		log.Info().Str("scope", "prepare_nodes").Int("nodes_num", len(nodesSet)).Msg("Preparing nodes")
	}
	st := time.Now()
	for nodeID := range nodesSet {
		node := nodesSet[nodeID]
		if node.useCount >= 2 || node.controlType == types.CONTROL_TYPE_IS_SIGNAL {
			node.isCrossing = true
		}
	}
	preparedNodes := make(map[osm.NodeID]*Node)
	// Filter nodes that are not used at all (building, parks and etc.)
	for nodeID := range nodesSet {
		node := nodesSet[nodeID]
		if node.useCount > 0 {
			preparedNodes[nodeID] = node
		}
	}
	if VERBOSE {
		log.Info().Str("scope", "prepare_nodes").Int("prepared_ways_num", len(preparedNodes)).Float64("elapsed", time.Since(st).Seconds()).Msg("Preparing nodes done!")
	}
	return preparedNodes, nil
}

// prepareWays prepares ways: link type, link class, link connection type, allowed agent types. Also mutates nodes data: increments use count (when being used in ways)
func prepareWays(ways []*WayOSM, nodesSet map[osm.NodeID]*Node, allowedAgentTypes []types.AgentType) ([]*WayOSM, error) {
	if VERBOSE {
		log.Info().Str("scope", "prepare_ways").Int("ways_num", len(ways)).Msg("Preparing ways")
	}
	st := time.Now()

	preparedWays := make([]*WayOSM, 0, len(ways))
	waysPOI := make([]*WayOSM, 0, len(ways)/2)
	for i := range ways {
		way := ways[i]
		if way.tags.IsPOI() {
			waysPOI = append(waysPOI, way)
			continue
		}

		nodesNum := len(way.Nodes)
		if nodesNum < 2 {
			log.Warn().Str("scope", "prepare_ways").Any("osm_way_id", way.ID).Int("nodes", nodesNum).Msg("Unexpected number of nodes")
			return preparedWays, nil
		}
		way.osmSourceNodeID = way.Nodes[0]
		way.osmTargetNodeID = way.Nodes[len(way.Nodes)-1]
		if way.osmSourceNodeID == way.osmTargetNodeID {
			way.isCycle = true
		}
		switch way.wayType {
		case WAY_TYPE_HIGHWAY:
			if way.wayPOI != nil {
				log.Warn().Str("scope", "prepare_ways").Any("osm_way_id", way.ID).Int("nodes", nodesNum).Msg("'highway' POI is not handled yet")
			}
			if way.isArea || way.isHighwayNegligible {
				continue
			}
			highwayType := types.NewHighwayTypeFrom(way.tags.Highway)
			linkInfo := types.NewCompositionLinkType(highwayType)
			if way.tags.OnewayDefault {
				// Override `oneway` for Way, but do not mutate source tags map
				way.isOneWay = types.NewOnewayDefault(linkInfo.LinkType)
			}
			way.linkConnectionType = linkInfo.LinkConnectionType
			way.linkType = linkInfo.LinkType
			way.linkClass = types.LINK_CLASS_HIGHWAY

			// Need to consider allowed tags only
			extractedAgentTypes := types.NewAllowableAgentTypeFrom(way.tags.MotorVehicle, way.tags.Motorcar, way.tags.Bicycle, way.tags.Foot, way.tags.Highway, way.tags.Access, way.tags.Service)
			agentsIntersection := types.AgentsIntersection(extractedAgentTypes, allowedAgentTypes)
			if len(agentsIntersection) == 0 {
				continue
			}
			way.allowedAgentTypes = make([]types.AgentType, 0, len(agentsIntersection))
			for agentType := range agentsIntersection {
				way.allowedAgentTypes = append(way.allowedAgentTypes, agentType)
			}
			// Increment nodes uses
			for _, nodeID := range way.Nodes {
				existingNode, ok := nodesSet[nodeID]
				if !ok {
					log.Warn().Str("scope", "prepare_ways").Any("osm_way_id", way.ID).Int("node_id", int(nodeID)).Msg("Can't find way node in nodes set")
					return preparedWays, nil
				}
				existingNode.useCount++
			}
			// Mark first and last node as used in cross
			nodesSet[way.Nodes[0]].isCrossing = true
			nodesSet[way.Nodes[len(way.Nodes)-1]].isCrossing = true
			// Append processed way to the filtered list
			preparedWays = append(preparedWays, way)
		case WAY_TYPE_RAILWAY:
			log.Warn().Str("scope", "prepare_ways").Any("osm_way_id", way.ID).Int("nodes", nodesNum).Msg("'railway' is not handled yet")
			if way.wayPOI != nil && way.wayPOI.poiType == POI_TYPE_RAILWAY {
				log.Warn().Str("scope", "prepare_ways").Any("osm_way_id", way.ID).Int("nodes", nodesNum).Msg("'railway' POI is not handled yet")
			}
		case WAY_TYPE_AEROWAY:
			log.Warn().Str("scope", "prepare_ways").Any("osm_way_id", way.ID).Int("nodes", nodesNum).Msg("'airway' is not handled yet")
			if way.wayPOI != nil && way.wayPOI.poiType == POI_TYPE_AEROWAY {
				log.Warn().Str("scope", "prepare_ways").Any("osm_way_id", way.ID).Int("nodes", nodesNum).Msg("'aeroway' POI is not handled yet")
			}
		default:
			// Just skip such way
		}
	}
	if VERBOSE {
		log.Info().Str("scope", "prepare_ways").Int("prepared_ways_num", len(preparedWays)).Float64("elapsed", time.Since(st).Seconds()).Msg("Preparing ways done!")
	}

	return preparedWays, nil
}
