package osm2gmns

import (
	"github.com/LdDl/go-gmns/gmns/types"
	"github.com/LdDl/osm2gmns/wrappers"
	"github.com/paulmach/osm"
)

type OSMScanner interface {
	Scan() bool
	Close() error
	Err() error
	Object() osm.Object
}

type OSMWaysNodes struct {
	Nodes map[osm.NodeID]*wrappers.NodeOSM
	Ways  []*wrappers.WayOSM

	AllowedAgentTypes []types.AgentType
}
