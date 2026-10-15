package preparation

import (
	"math"
	"testing"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/go-gmns/gmns/types"
	"github.com/LdDl/go-gmns/macro"
	"github.com/LdDl/go-gmns/movement"
	"github.com/stretchr/testify/assert"
)

func TestCustomUseParametersAndSuppliedDefinitions(t *testing.T) {
	net := macro.NewNet()
	assert.NoError(t, net.SetConfig(gmns.NewConfig(gmns.IDTypeInteger)))
	first := gmns.NodeIDFromInt64(0)
	last := gmns.NodeIDFromInt64(-1)
	road := gmns.LinkIDFromInt64(0)
	net.Nodes[first] = macro.NewNodeFrom(first)
	net.Nodes[last] = macro.NewNodeFrom(last)
	agents := []types.AgentType{types.AGENT_BIKE, types.AGENT_AUTO}
	net.Links[road] = macro.NewLinkFrom(road, first, last, macro.WithAllowedAgentTypes(agents))
	supplied := gmns.NewUseDefinition("bike", 2, 0.75)
	assert.NoError(t, net.AddUseDefinition(supplied))
	options := DefaultUseOptions()
	options.AutoPersonsPerVehicle = 1.6
	assert.NoError(t, Macro(net, options))
	assert.Equal(t, "auto,bike", net.Links[road].AllowedUses())
	auto, found := net.UseDefinition("auto")
	assert.True(t, found)
	assert.Equal(t, 1.6, auto.PersonsPerVehicle())
	bike, found := net.UseDefinition("bike")
	assert.True(t, found)
	assert.Equal(t, 0.75, bike.PCE())
	assert.NoError(t, Macro(net, options), "preparation is repeatable")
	options.AutoPCE = math.NaN()
	assert.Error(t, Macro(net, options))
}

func TestGeneratedUTurnKeepsTypeAndOriginalCode(t *testing.T) {
	net := macro.NewNet()
	nodeID := gmns.NodeIDFromInt64(0)
	linkID := gmns.LinkIDFromInt64(0)
	identifier := gmns.MovementIDFromInt64(0)
	record := movement.NewMovement(identifier, nodeID, linkID, linkID, movement.MOVEMENT_EBU, movement.MOVEMENT_TYPE_U_TURN)
	records := movement.MovementsStorage{identifier: record}
	codes, err := StoreGeneratedMovements(net, records)
	assert.NoError(t, err)
	assert.Equal(t, "EBU", codes["0"])
	stored, found := net.Movement(identifier)
	assert.True(t, found)
	assert.Empty(t, stored.MvmtTextID())
	assert.Equal(t, movement.MOVEMENT_TYPE_U_TURN, stored.Type())
	assert.Equal(t, movement.MOVEMENT_EBU, record.MvmtTextID(), "generator results must remain intact")
}
