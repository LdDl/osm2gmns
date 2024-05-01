package osm2gmns

import (
	"time"

	"github.com/rs/zerolog"
)

func init() {
	zerolog.TimeFieldFormat = time.RFC3339
}

var (
	VERBOSE = true
)
