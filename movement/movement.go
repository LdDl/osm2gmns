package movement

import (
	"sync"

	"github.com/LdDl/osm2gmns/geomath"
	"github.com/LdDl/osm2gmns/gmns"
	"github.com/paulmach/orb"
)

type autoInc struct {
	sync.Mutex
	id MovementID
}

func (a *autoInc) ID() (id MovementID) {
	a.Lock()
	defer a.Unlock()

	id = a.id
	a.id++
	return
}

var (
	ai autoInc
)

type MovementID int

type Movement struct {
	ID                 MovementID
	MacroNodeID        gmns.NodeID
	IncomeMacroLinkID  gmns.LinkID
	OutcomeMacroLinkID gmns.LinkID
	Geom               orb.LineString
	GeomEuclidean      orb.LineString
}

// NewMovement constructs new movement;
// macroNodeID - source of movement;
// incomeMacroLinkID - identifier of macro link represents source of movement;
// outcomeMacroLinkID - identifier of  macro link represents target of movement;
// mvmtTxtID - composite type of movement. One of: SBT, SBR, SBL, SBU, EBT, EBR, EBL, EBU, NBT, NBR, NBL, NBU, WBT, WBR, WBL, WBU;
// mvmtType - type of movement. One of: THRU, RIGHT, LEFT, U_TURN;
// geom - geometry for the Movement in EPSG:4326. It will prepares EPSG:3857 automatically.
func NewMovement(macroNodeID gmns.NodeID, incomeMacroLinkID, outcomeMacroLinkID gmns.LinkID, mvmtTxtID MovementCompositeType, mvmtType MovementType, geom orb.LineString, options ...func(*Movement)) *Movement {
	mvmt := &Movement{
		ID:                 ai.ID(),
		MacroNodeID:        macroNodeID,
		IncomeMacroLinkID:  incomeMacroLinkID,
		OutcomeMacroLinkID: outcomeMacroLinkID,
		Geom:               geom,
		GeomEuclidean:      geomath.LineToEuclidean(geom),
	}
	for _, o := range options {
		o(mvmt)
	}
	return mvmt
}
