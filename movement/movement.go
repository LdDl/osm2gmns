package movement

import (
	"sync"
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
	ID MovementID
}

func NewMovement() *Movement {
	mvmt := &Movement{
		ID: ai.ID(),
	}
	return mvmt
}
