package osm2gmns

import (
	"github.com/paulmach/osm"
)

type Way struct {
	Nodes  osm.WayNodes
	TagMap osm.Tags
	ID     osm.WayID
	Oneway bool
}
