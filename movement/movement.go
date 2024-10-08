package movement

import (
	"sync"

	"github.com/LdDl/osm2gmns/geomath"
	"github.com/LdDl/osm2gmns/gmns"
	"github.com/LdDl/osm2gmns/types"
	"github.com/paulmach/orb"
	"github.com/paulmach/osm"
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
	name string

	ID          MovementID
	MType       MovementType
	MTextID     MovementCompositeType
	osmNodeID   osm.NodeID
	controlType types.ControlType
	lanesNum    int

	MacroNodeID                              gmns.NodeID
	IncomeMacroLinkID                        gmns.LinkID
	fromOsmNodeID                            osm.NodeID
	startIncomeLaneSeqID, endIncomeLaneSeqID int
	incomeLaneStart, incomeLaneEnd           int

	OutcomeMacroLinkID                         gmns.LinkID
	toOsmNodeID                                osm.NodeID
	startOutcomeLaneSeqID, endOutcomeLaneSeqID int
	outcomeLaneStart, outcomeLaneEnd           int

	Geom          orb.LineString
	GeomEuclidean orb.LineString

	allowedAgentTypes []types.AgentType
}

// NewMovement constructs new movement;
// macroNodeID - source of movement;
// incomeMacroLinkID - identifier of macro link represents source of movement;
// outcomeMacroLinkID - identifier of  macro link represents target of movement;
// mvmtTxtID - composite type of movement. One of: SBT, SBR, SBL, SBU, EBT, EBR, EBL, EBU, NBT, NBR, NBL, NBU, WBT, WBR, WBL, WBU;
// mvmtType - type of movement. One of: THRU, RIGHT, LEFT, U_TURN;
// geom - geometry for the Movement in EPSG:4326. It will prepares EPSG:3857 automatically.
func NewMovement(macroNodeID gmns.NodeID, incomeMacroLinkID, outcomeMacroLinkID gmns.LinkID, mvmtTxtID MovementCompositeType, mvmtType MovementType, geom orb.LineString, options ...func(*Movement)) Movement {
	mvmt := Movement{
		name:               "-",
		ID:                 ai.ID(),
		MType:              mvmtType,
		MTextID:            mvmtTxtID,
		MacroNodeID:        macroNodeID,
		IncomeMacroLinkID:  incomeMacroLinkID,
		OutcomeMacroLinkID: outcomeMacroLinkID,
		Geom:               geom,
		GeomEuclidean:      geomath.LineToEuclidean(geom),
	}
	for _, o := range options {
		o(&mvmt)
	}
	return mvmt
}

// WithAllowedAgentTypes sets agent types which are allowed to use given movement
// Notice: it copies given agent types slice
func WithAllowedAgentTypes(agentTypes []types.AgentType) func(*Movement) {
	return func(mvmt *Movement) {
		mvmt.allowedAgentTypes = make([]types.AgentType, len(agentTypes))
		copy(mvmt.allowedAgentTypes, agentTypes)
	}
}

// WithControlType sets control type
// Notice: you should provide control type same as for underlying macro node
func WithControlType(ctrlType types.ControlType) func(*Movement) {
	return func(mvmt *Movement) {
		mvmt.controlType = ctrlType
	}
}

// WithOSMNode sets OSM node ID for the movement
// Notice: you should provide OSM node ID for underlying macro node
func WithOSMNode(osmNodeID osm.NodeID) func(*Movement) {
	return func(mvmt *Movement) {
		mvmt.osmNodeID = osmNodeID
	}
}

// WithSourceOSMNode sets source OSM node ID for the movement
// Notice: you should provide source OSM node ID for underlying income macro link
func WithSourceOSMNode(osmNodeID osm.NodeID) func(*Movement) {
	return func(mvmt *Movement) {
		mvmt.fromOsmNodeID = osmNodeID
	}
}

// WithTargetOSMNode sets target OSM node ID for the movement
// Notice: you should provide target OSM node ID for underlying outcome macro link
func WithTargetOSMNode(osmNodeID osm.NodeID) func(*Movement) {
	return func(mvmt *Movement) {
		mvmt.toOsmNodeID = osmNodeID
	}
}

// WithLanesNum sets number of lanes in the movement
// Notice: you should provide number as (incomeLaneIndexEnd - incomeLaneIndexStart + 1)
func WithLanesNum(n int) func(*Movement) {
	return func(mvmt *Movement) {
		mvmt.lanesNum = n
	}
}

// WithIncomeLane sets start and end index for the lane of income macro link
func WithIncomeLane(start int, end int) func(*Movement) {
	return func(mvmt *Movement) {
		mvmt.incomeLaneStart = start
		mvmt.incomeLaneEnd = end
	}
}

// WithIncomeLaneSequence sets start and end index for the lane's segment of income macro link
// Notice: those are just indexes of incomeLaneStart and incomeLaneEnd
func WithIncomeLaneSequence(startIdx int, endIdx int) func(*Movement) {
	return func(mvmt *Movement) {
		mvmt.startIncomeLaneSeqID = startIdx
		mvmt.endIncomeLaneSeqID = endIdx
	}
}

// WithOutcomeLane sets start and end index for the lane of outcome macro link
func WithOutcomeLane(start int, end int) func(*Movement) {
	return func(mvmt *Movement) {
		mvmt.outcomeLaneStart = start
		mvmt.outcomeLaneEnd = end
	}
}

// WithOutcomeLaneSequence sets start and end index for the lane's segment of outcome macro link
// Notice: those are just indexes of outcomeLaneStart and outcomeLaneEnd
func WithOutcomeLaneSequence(startIdx int, endIdx int) func(*Movement) {
	return func(mvmt *Movement) {
		mvmt.startOutcomeLaneSeqID = startIdx
		mvmt.endOutcomeLaneSeqID = endIdx
	}
}

// WithName sets alias for the movement
func WithName(name string) func(*Movement) {
	return func(mvmt *Movement) {
		mvmt.name = name
	}
}
