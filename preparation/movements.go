package preparation

import (
	"fmt"
	"sort"
	"strings"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/go-gmns/macro"
	"github.com/LdDl/go-gmns/movement"
	"github.com/LdDl/osm2gmns/csvexport"
)

// StoreGeneratedMovements stores standard records after generation, retaining
// generator-only codes separately for application provenance.
func StoreGeneratedMovements(net *macro.Net, records movement.MovementsStorage) (map[string]string, error) {
	if net == nil {
		return nil, fmt.Errorf("nil macro network")
	}
	identifiers := make([]gmns.MovementID, 0, len(records))
	for id, record := range records {
		err := record.Validate()
		if err != nil {
			return nil, err
		}
		if record.ID != id {
			return nil, fmt.Errorf("movement %v: storage key differs from record ID %v", id, record.ID)
		}
		identifiers = append(identifiers, id)
	}
	sort.Slice(identifiers, func(first, second int) bool {
		return identifiers[first].Less(identifiers[second])
	})
	originalCodes := make(map[string]string)
	for _, id := range identifiers {
		record := records[id]
		prepared := record.Clone()
		code := prepared.MvmtTextID().String()
		if !csvexport.ValidMovementCode(code) {
			isUTurn := prepared.Type() == movement.MOVEMENT_TYPE_U_TURN && strings.HasSuffix(code, "U")
			if !isUTurn && code != "undefined" {
				return nil, fmt.Errorf("movement %v: unsupported generated code %q", record.ID, code)
			}
			originalCodes[record.ID.String()] = code
			movement.WithMvmtTextID("")(prepared)
		}
		err := net.AddMovement(prepared)
		if err != nil {
			return nil, err
		}
	}
	return originalCodes, nil
}
