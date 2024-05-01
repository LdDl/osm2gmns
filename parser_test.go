package osm2gmns

import (
	"testing"

	"github.com/LdDl/osm2gmns/types"
)

func TestParser(t *testing.T) {
	parser := NewParser(
		"./sample.osm",
		WithPreparePOI(false),
		WithStrictMode(false),
		WithVerbose(true),
		WithAllowedAgentTypes(types.AGENT_TYPES_DEFAULT),
	)
	t.Log(parser)

	osmData, err := readOSM(parser.filename, parser.preparePOI)
	if err != nil {
		t.Error(err)
		return
	}

	err = osmData.prepareNetwork(parser.allowedAgentTypes, parser.preparePOI)
	if err != nil {
		t.Error(err)
		return
	}
	// t.Error(0)
}
