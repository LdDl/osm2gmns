package preparation

import (
	"fmt"

	"github.com/LdDl/go-gmns/micro"
	"github.com/LdDl/go-gmns/movement"
)

// Micro prepares standard uses on a generated network after all graph generators have finished.
// Existing supplied use references and definitions are preserved.
func Micro(net *micro.Net, options *UseOptions) error {
	if net == nil {
		return fmt.Errorf("nil micro network")
	}
	err := net.ValidateDataset()
	if err != nil {
		return err
	}
	definitions, err := options.Definitions()
	if err != nil {
		return err
	}
	for _, definition := range definitions {
		_, exists := net.UseDefinition(definition.Use())
		if !exists {
			err = net.AddUseDefinition(definition)
			if err != nil {
				return err
			}
		}
	}
	for _, link := range net.Links {
		if link.AllowedUses() != "" {
			continue
		}
		uses, useErr := agentUses(link.AllowedAgentTypes())
		if useErr != nil {
			return useErr
		}
		micro.WithAllowedUses(uses)(link)
	}
	for _, record := range net.MovementRecords() {
		if record.AllowedUses() != "" {
			continue
		}
		uses, useErr := agentUses(record.AllowedAgentTypes())
		if useErr != nil {
			return useErr
		}
		movement.WithAllowedUses(uses)(record)
		err = net.UpdateMovement(record)
		if err != nil {
			return err
		}
	}
	return net.ValidateDataset()
}
