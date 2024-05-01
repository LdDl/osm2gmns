package osm2gmns

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/LdDl/osm2gmns/types"
	"github.com/LdDl/osm2gmns/wrappers"
	"github.com/paulmach/osm"
	"github.com/paulmach/osm/osmpbf"
	"github.com/paulmach/osm/osmxml"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

func readOSM(filename string, poi bool) (*OSMWaysNodes, error) {
	if VERBOSE {
		log.Info().Str("scope", "osm_read").Str("filename", filename).Msg("Opening file")
	}
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	/* Process ways */
	if VERBOSE {
		log.Info().Str("scope", "osm_read").Msg("Processing ways")
	}
	st := time.Now()

	ways := []*WayOSM{}
	nodesSeen := make(map[osm.NodeID]struct{})
	{
		var scannerWays OSMScanner

		// Guess file extension and prepare correct scanner for ways
		ext := filepath.Ext(filename)
		switch ext {
		case ".osm", ".xml":
			scannerWays = osmxml.New(context.Background(), file)
		case ".pbf", ".osm.pbf":
			scannerWays = osmpbf.New(context.Background(), file, 4)
		default:
			return nil, fmt.Errorf("file extension '%s' for file '%s' is not handled yet", ext, filename)
		}
		defer scannerWays.Close()

		// Scan ways
		for scannerWays.Scan() {
			obj := scannerWays.Object()
			if obj.ObjectID().Type() != "way" {
				continue
			}
			way := obj.(*osm.Way)
			// Call tags flattening to make further processing easier
			tags := wrappers.NewWayTagsFrom(way)

			wayType := WAY_TYPE_UNDEFINED
			poiName := ""
			poiType := POI_TYPE_UNDEFINED
			if tags.IsHighway() {
				wayType = WAY_TYPE_HIGHWAY
				if tags.IsHighwayPOI() {
					poiName = tags.Highway
					poiType = POI_TYPE_HIGHWAY
				}
			} else if tags.IsRailway() {
				wayType = WAY_TYPE_RAILWAY
				if tags.IsRailwayPOI() {
					poiName = tags.Railway
					poiType = POI_TYPE_RAILWAY
				}
			} else if tags.IsAeroway() {
				wayType = WAY_TYPE_AEROWAY
				if tags.IsAerowayPOI() {
					poiName = tags.Aeroway
					poiType = POI_TYPE_AEROWAY
				}
			}

			preparedWay := &WayOSM{
				ID:                  way.ID,
				wayType:             wayType,
				isHighwayNegligible: tags.IsHighwayNegligible(),
				Nodes:               make([]osm.NodeID, 0, len(way.Nodes)),
				tags:                tags,
				freeSpeed:           -1.0,
				capacity:            -1.0,
				isArea:              tags.Area != "" && tags.Area != "no",
				isOneWay:            tags.Oneway,
			}
			if poiName != "" {
				preparedWay.wayPOI = &WayPOIProps{
					poiName: poiName,
					poiType: poiType,
				}
			}
			// Mark way's nodes as seen to remove isolated nodes in further
			for _, node := range way.Nodes {
				nodesSeen[node.ID] = struct{}{}
				preparedWay.Nodes = append(preparedWay.Nodes, node.ID)
			}
			ways = append(ways, preparedWay)
		}
		err = scannerWays.Err()
		if err != nil {
			return nil, err
		}
	}

	if VERBOSE {
		// fmt.Printf("Done in %v\n", time.Since(st))
		log.Info().Str("scope", "osm_read").Float64("elapsed", time.Since(st).Seconds()).Msg("Processing ways done!")
	}
	// Seek file to start
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return nil, errors.Wrap(err, "Can't repeat seeking after ways scanning")
	}

	/* Process nodes */
	if VERBOSE {
		log.Info().Str("scope", "osm_read").Msg("Processing nodes")
	}
	st = time.Now()
	nodes := make(map[osm.NodeID]*Node)
	{

		var scannerNodes OSMScanner

		// Guess file extension and prepare correct scanner for ways
		ext := filepath.Ext(filename)
		switch ext {
		case ".osm", ".xml":
			scannerNodes = osmxml.New(context.Background(), file)
		case ".pbf", ".osm.pbf":
			scannerNodes = osmpbf.New(context.Background(), file, 4)
		default:
			return nil, fmt.Errorf("file extension '%s' for file '%s' is not handled yet", ext, filename)
		}
		defer scannerNodes.Close()

		// Scan nodes
		for scannerNodes.Scan() {
			obj := scannerNodes.Object()
			if obj.ObjectID().Type() != "node" {
				continue
			}
			node := obj.(*osm.Node)
			if _, ok := nodesSeen[node.ID]; ok {
				delete(nodesSeen, node.ID)
				nameText := node.Tags.Find("name")
				highwayText := node.Tags.Find("highway")
				controlType := types.CONTROL_TYPE_NOT_SIGNAL
				if highwayText == "traffic_signals" {
					controlType = types.CONTROL_TYPE_IS_SIGNAL
				}
				nodes[node.ID] = &Node{
					name:        nameText,
					node:        *node,
					ID:          node.ID,
					useCount:    0,
					isCrossing:  false,
					controlType: controlType,
					osmData: NodeOSMInfo{
						highway: highwayText,
					},
				}
			}
		}
		err = scannerNodes.Err()
		if err != nil {
			return nil, err
		}
	}

	if VERBOSE {
		log.Info().Str("scope", "osm_read").Float64("elapsed", time.Since(st).Seconds()).Msg("Processing nodes done!")
	}

	if VERBOSE {
		log.Info().Str("scope", "osm_read").Int("ways_num", len(ways)).Msg("")
		log.Info().Str("scope", "osm_read").Int("nodes_num", len(nodes)).Msg("")
	}
	return &OSMWaysNodes{
		ways:  ways,
		nodes: nodes,
	}, nil
}
