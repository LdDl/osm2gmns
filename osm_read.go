package osm2gmns

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/LdDl/go-gmns/gmns/types"
	"github.com/LdDl/osm2gmns/wrappers"
	"github.com/paulmach/osm"
	"github.com/paulmach/osm/osmpbf"
	"github.com/paulmach/osm/osmxml"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

var (
	ErrBadFileExtension = fmt.Errorf("bad file extension")
)

func GuessParserType(filename string) (ParserType, error) {
	ext := filepath.Ext(filename)
	switch ext {
	case ".osm", ".xml":
		return PARSER_XML, nil
	case ".pbf", ".osm.pbf":
		return PARSER_PBF, nil
	default:
		return PARSE_UNDEFINED, errors.Wrapf(ErrBadFileExtension, "Filename: '%s', Extension: '%s'", filename, ext)
	}
}

func (parser *Parser) ReadOSM() (*OSMWaysNodes, error) {
	filename, poi := parser.filename, parser.preparePOI
	_ = poi
	if VERBOSE {
		log.Info().Str("scope", "osm_read").Str("filename", filename).Msg("Opening file")
	}

	parserType, err := GuessParserType(filename)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return ReadOSMFromFile(file, parserType, parser.allowedAgentTypes)
}

func ReadOSMFromFile(file *os.File, parserType ParserType, allowedAgentTypes []types.AgentType) (*OSMWaysNodes, error) {
	/* Process ways */
	if VERBOSE {
		log.Info().Str("scope", "osm_read").Msg("Processing ways")
	}
	st := time.Now()

	ways := []*wrappers.WayOSM{}
	nodesSeen := make(map[osm.NodeID]struct{})
	{
		var scannerWays OSMScanner
		bufReader := bufio.NewReaderSize(file, 128*1024*1024)
		// Guess file extension and prepare correct scanner for ways
		switch parserType {
		case PARSER_XML:
			scannerWays = osmxml.New(context.Background(), bufReader)
		case PARSER_PBF:
			scannerWays = osmpbf.New(context.Background(), file, 4)
		default:
			return nil, fmt.Errorf("file extension '%s' is not handled yet", parserType)
		}
		defer scannerWays.Close()

		// Scan ways
		for scannerWays.Scan() {
			obj := scannerWays.Object()
			if obj.ObjectID().Type() != "way" {
				continue
			}
			way := obj.(*osm.Way)
			preparedWay := wrappers.NewWayOSMFrom(way)
			// Mark way's nodes as seen to remove isolated nodes in further
			for _, node := range way.Nodes {
				nodesSeen[node.ID] = struct{}{}
			}
			ways = append(ways, preparedWay)
		}
		err := scannerWays.Err()
		if err != nil {
			return nil, err
		}
	}

	if VERBOSE {
		log.Info().Str("scope", "osm_read").Float64("elapsed", time.Since(st).Seconds()).Msg("Processing ways done!")
	}
	// Seek file to start
	_, err := file.Seek(0, io.SeekStart)
	if err != nil {
		return nil, errors.Wrap(err, "Can't repeat seeking after ways scanning")
	}

	/* Process nodes */
	if VERBOSE {
		log.Info().Str("scope", "osm_read").Msg("Processing nodes")
	}
	st = time.Now()
	nodes := make(map[osm.NodeID]*wrappers.NodeOSM)
	{

		var scannerNodes OSMScanner

		// Guess file extension and prepare correct scanner for ways
		switch parserType {
		case PARSER_XML:
			scannerNodes = osmxml.New(context.Background(), file)
		case PARSER_PBF:
			scannerNodes = osmpbf.New(context.Background(), file, 4)
		default:
			return nil, fmt.Errorf("file extension '%s' is not handled yet", parserType)
		}
		defer scannerNodes.Close()

		// Scan nodes
		for scannerNodes.Scan() {
			obj := scannerNodes.Object()
			if obj.ObjectID().Type() != "node" {
				continue
			}
			node := obj.(*osm.Node)
			// if _, ok := nodesSeen[node.ID]; ok {
			// delete(nodesSeen, node.ID)
			preparedNode := wrappers.NewNodeOSMFrom(node)
			nodes[node.ID] = preparedNode
			// }
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

	osmData := &OSMWaysNodes{
		ways:              ways,
		nodes:             nodes,
		allowedAgentTypes: make([]types.AgentType, len(allowedAgentTypes)),
	}
	copy(osmData.allowedAgentTypes, allowedAgentTypes)
	return osmData, nil
}
