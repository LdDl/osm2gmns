package preparation

import (
	"fmt"
	"sort"
	"strings"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/go-gmns/gmns/types"
)

const (
	DefaultAutoPersonsPerVehicle = 1.0
	DefaultAutoPCE               = 1.0
	DefaultBikePersonsPerVehicle = 1.0
	DefaultBikePCE               = 0.5
	DefaultWalkPersonsPerVehicle = 1.0
	DefaultWalkPCE               = 0.0
)

// UseOptions defines modeling assumptions, not occupancy measured from OSM.
type UseOptions struct {
	AutoPersonsPerVehicle float64
	AutoPCE               float64
	BikePersonsPerVehicle float64
	BikePCE               float64
	WalkPersonsPerVehicle float64
	WalkPCE               float64
}

func DefaultUseOptions() *UseOptions {
	return &UseOptions{
		AutoPersonsPerVehicle: DefaultAutoPersonsPerVehicle,
		AutoPCE:               DefaultAutoPCE,
		BikePersonsPerVehicle: DefaultBikePersonsPerVehicle,
		BikePCE:               DefaultBikePCE,
		WalkPersonsPerVehicle: DefaultWalkPersonsPerVehicle,
		WalkPCE:               DefaultWalkPCE,
	}
}

func (options *UseOptions) Definitions() ([]*gmns.UseDefinition, error) {
	if options == nil {
		return nil, fmt.Errorf("use options are required")
	}
	description := gmns.WithUseDescription("Converter modeling parameters; not measured from OSM")
	records := []*gmns.UseDefinition{
		gmns.NewUseDefinition("auto", options.AutoPersonsPerVehicle, options.AutoPCE, description),
		gmns.NewUseDefinition("bike", options.BikePersonsPerVehicle, options.BikePCE, description),
		gmns.NewUseDefinition("walk", options.WalkPersonsPerVehicle, options.WalkPCE, description),
	}
	for _, record := range records {
		err := record.Validate()
		if err != nil {
			return nil, fmt.Errorf("use %s: %w", record.Use(), err)
		}
	}
	return records, nil
}

func agentUses(agents []types.AgentType) (string, error) {
	names := make([]string, 0, len(agents))
	seen := make(map[types.AgentType]bool)
	for _, agent := range agents {
		if agent != types.AGENT_AUTO && agent != types.AGENT_BIKE && agent != types.AGENT_WALK {
			return "", fmt.Errorf("unsupported agent value %d", agent)
		}
		if !seen[agent] {
			names = append(names, agent.String())
			seen[agent] = true
		}
	}
	sort.Strings(names)
	return strings.Join(names, ","), nil
}
