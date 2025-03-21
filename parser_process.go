package osm2gmns

import (
	"time"

	"github.com/LdDl/go-gmns/macro"

	"github.com/LdDl/go-gmns/gmns/types"
	"github.com/LdDl/osm2gmns/osmmacro"
	"github.com/LdDl/osm2gmns/wrappers"
	"github.com/paulmach/osm"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

func GenerateMacroscopic(osmData *OSMWaysNodes, poi bool) (*macro.Net, error) {
	ways, nodesSet, allowedAgentTypes := osmData.Ways, osmData.Nodes, osmData.AllowedAgentTypes
	preparedWays, err := prepareWays(ways, nodesSet, allowedAgentTypes)
	if err != nil {
		return nil, errors.Wrap(err, "Can't prepare ways")
	}
	preparedNodes, err := prepareNodes(nodesSet)
	if err != nil {
		return nil, errors.Wrap(err, "Can't prepare nodes")
	}
	err = markPureCycles(preparedNodes, preparedWays)
	if err != nil {
		return nil, errors.Wrap(err, "Can't mark pure cycles")
	}
	if VERBOSE {
		log.Info().Str("scope", "gen_macro").Msg("Preparing macroscopic network")
	}
	macroNet, err := osmmacro.NewNetFromOSM(preparedWays, preparedNodes)
	if err != nil {
		return nil, errors.Wrap(err, "Can't prepare macroscopic network")
	}
	if VERBOSE {
		log.Info().Str("scope", "gen_macro").Int("macro_nodes_num", len(macroNet.Nodes)).Int("macro_links_num", len(macroNet.Links)).Msg("Preparing macroscopic network done!")
	}
	return macroNet, nil
}

// prepareNodes examines nodes which has use count > 0 and use count > 2 on being cross
func prepareNodes(nodesSet map[osm.NodeID]*wrappers.NodeOSM) (map[osm.NodeID]*wrappers.NodeOSM, error) {
	if VERBOSE {
		log.Info().Str("scope", "prepare_nodes").Int("nodes_num", len(nodesSet)).Msg("Preparing nodes")
	}
	st := time.Now()
	for nodeID := range nodesSet {
		node := nodesSet[nodeID]
		if node.UseCount >= 2 || node.ControlType == types.CONTROL_TYPE_IS_SIGNAL {
			node.IsCrossing = true
		}
	}
	preparedNodes := make(map[osm.NodeID]*wrappers.NodeOSM)
	// Filter nodes that are not used at all (building, parks and etc.)
	for nodeID := range nodesSet {
		node := nodesSet[nodeID]
		if node.UseCount > 0 {
			preparedNodes[nodeID] = node
		}
	}
	if VERBOSE {
		log.Info().Str("scope", "prepare_nodes").Int("prepared_nodes_num", len(preparedNodes)).Float64("elapsed", time.Since(st).Seconds()).Msg("Preparing nodes done!")
	}
	return preparedNodes, nil
}

// prepareWays prepares ways: link type, link class, link connection type, allowed agent types. Also mutates nodes data: increments use count (when being used in ways)
func prepareWays(ways []*wrappers.WayOSM, nodesSet map[osm.NodeID]*wrappers.NodeOSM, allowedAgentTypes []types.AgentType) ([]*wrappers.WayOSM, error) {
	if VERBOSE {
		log.Info().Str("scope", "prepare_ways").Int("ways_num", len(ways)).Msg("Preparing ways")
	}
	st := time.Now()

	preparedWays := make([]*wrappers.WayOSM, 0, len(ways))
	waysPOI := make([]*wrappers.WayOSM, 0, len(ways)/2)
	badArea := 0
	badHighway := 0
	badAgents := 0
	badNodesNum := 0
	for i := range ways {
		way := ways[i]
		if way.Tags.IsPOI() {
			waysPOI = append(waysPOI, way)
			continue
		}

		nodesNum := len(way.Nodes)
		if nodesNum < 2 {
			badNodesNum++
			log.Warn().Str("scope", "prepare_ways").Any("osm_way_id", way.ID).Int("nodes", nodesNum).Msg("Unexpected number of nodes")
			return preparedWays, nil
		}
		way.OsmSourceNodeID = way.Nodes[0]
		way.OsmTargetNodeID = way.Nodes[len(way.Nodes)-1]
		if way.OsmSourceNodeID == way.OsmTargetNodeID {
			way.IsCycle = true
		}
		switch way.WayType {
		case wrappers.WAY_TYPE_HIGHWAY:
			if way.WayPOI != nil {
				log.Warn().Str("scope", "prepare_ways").Any("osm_way_id", way.ID).Int("nodes", nodesNum).Msg("'highway' POI is not handled yet")
				continue
			}
			if way.IsArea {
				badArea++
				continue
			}
			if way.IsHighwayNegligible {
				badHighway++
				continue
			}
			highwayType := types.NewHighwayTypeFrom(way.Tags.Highway)
			linkInfo := types.NewCompositionLinkType(highwayType)
			if linkInfo.LinkType == types.LINK_UNDEFINED && linkInfo.LinkConnectionType == types.NOT_A_LINK {
				continue
			}

			if way.Tags.OnewayDefault {
				// Override `oneway` for Way, but do not mutate source tags map
				way.IsOneWay = types.NewOnewayDefault(linkInfo.LinkType)
			}
			way.LinkConnectionType = linkInfo.LinkConnectionType
			way.LinkType = linkInfo.LinkType
			way.LinkClass = types.LINK_CLASS_HIGHWAY

			// Need to consider allowed tags only
			extractedAgentTypes := types.NewAllowableAgentTypeFrom(way.Tags.MotorVehicle, way.Tags.Motorcar, way.Tags.Bicycle, way.Tags.Foot, way.Tags.Highway, way.Tags.Access, way.Tags.Service)
			agentsIntersection := types.AgentsIntersection(extractedAgentTypes, allowedAgentTypes)
			if len(agentsIntersection) == 0 {
				badAgents++
				continue
			}
			way.AllowedAgentTypes = make([]types.AgentType, 0, len(agentsIntersection))
			for agentType := range agentsIntersection {
				way.AllowedAgentTypes = append(way.AllowedAgentTypes, agentType)
			}
			// Increment nodes uses
			for _, nodeID := range way.Nodes {
				existingNode, ok := nodesSet[nodeID]
				if !ok {
					log.Warn().Str("scope", "prepare_ways").Any("osm_way_id", way.ID).Int("node_id", int(nodeID)).Msg("Can't find way node in nodes set")
					return preparedWays, nil
				}
				existingNode.UseCount++
			}
			// Mark first and last node as used in cross
			nodesSet[way.Nodes[0]].IsCrossing = true
			nodesSet[way.Nodes[len(way.Nodes)-1]].IsCrossing = true
			// Append processed way to the filtered list
			preparedWays = append(preparedWays, way)
		case wrappers.WAY_TYPE_RAILWAY:
			log.Warn().Str("scope", "prepare_ways").Any("osm_way_id", way.ID).Int("nodes", nodesNum).Msg("'railway' is not handled yet")
			if way.WayPOI != nil && way.WayPOI.PoiType == types.POI_TYPE_RAILWAY {
				log.Warn().Str("scope", "prepare_ways").Any("osm_way_id", way.ID).Int("nodes", nodesNum).Msg("'railway' POI is not handled yet")
			}
		case wrappers.WAY_TYPE_AEROWAY:
			log.Warn().Str("scope", "prepare_ways").Any("osm_way_id", way.ID).Int("nodes", nodesNum).Msg("'airway' is not handled yet")
			if way.WayPOI != nil && way.WayPOI.PoiType == types.POI_TYPE_AEROWAY {
				log.Warn().Str("scope", "prepare_ways").Any("osm_way_id", way.ID).Int("nodes", nodesNum).Msg("'aeroway' POI is not handled yet")
			}
		default:
			// Just skip such way
		}
	}
	if badArea > 0 {
		log.Warn().Str("scope", "prepare_ways").Int("ways_num", badArea).Msg("Area ways")
	}
	if badHighway > 0 {
		log.Warn().Str("scope", "prepare_ways").Int("ways_num", badHighway).Msg("Unexpected highway tags")
	}
	if badAgents > 0 {
		log.Warn().Str("scope", "prepare_ways").Int("ways_num", badAgents).Msg("Ways with bad agents data")
	}
	if badNodesNum > 0 {
		log.Warn().Str("scope", "prepare_ways").Int("ways_num", badNodesNum).Msg("Ways with <2 nodes")
	}
	if VERBOSE {
		log.Info().Str("scope", "prepare_ways").Int("prepared_ways_num", len(preparedWays)).Float64("elapsed", time.Since(st).Seconds()).Msg("Preparing ways done!")
	}

	return preparedWays, nil
}
