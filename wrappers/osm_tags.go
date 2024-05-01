package wrappers

import (
	"regexp"
	"strconv"

	"github.com/paulmach/osm"
	"github.com/rs/zerolog/log"
)

var (
	junctionTypes = map[string]struct{}{
		"circular":   {},
		"roundabout": {},
	}

	poiHighwayTags = map[string]struct{}{
		"bus_stop": {},
		"platform": {},
	}

	poiRailwayTags = map[string]struct{}{
		"depot":         {},
		"workshop":      {},
		"halt":          {},
		"interlocking":  {},
		"junction":      {},
		"spur_junction": {},
		"terminal":      {},
		"platform":      {},
	}

	poiAerowayTags = map[string]struct{}{}

	negligibleHighwayTags = map[string]struct{}{
		"path":         {},
		"construction": {},
		"proposed":     {},
		"raceway":      {},
		"bridleway":    {},
		"rest_area":    {},
		"su":           {},
		"road":         {},
		"abandoned":    {},
		"planned":      {},
		"trailhead":    {},
		"stairs":       {},
		"dismantled":   {},
		"disused":      {},
		"razed":        {},
		"access":       {},
		"corridor":     {},
		"stop":         {},
	}

	// See ref.: https://wiki.openstreetmap.org/wiki/Tag:oneway%3Dreversible
	onewayReversible = map[string]struct{}{
		"reversible":  {},
		"alternating": {},
	}

	mphRegExp   = regexp.MustCompile(`\d+\.?\d* mph`)
	kmhRegExp   = regexp.MustCompile(`\d+\.?\d* km/h`)
	lanesRegExp = regexp.MustCompile(`\d+\.?\d*`)
)

type WayTags struct {
	name              string
	Highway           string
	Railway           string
	Aeroway           string
	turnLanes         string
	turnLanesForward  string
	turnLanesBackward string

	Area         string
	MotorVehicle string
	Access       string
	Motorcar     string
	Service      string
	Foot         string
	Bicycle      string
	building     string
	amenity      string
	leisure      string
	junction     string

	maxSpeed float64

	lanes         int
	lanesForward  int
	lanesBackward int

	Oneway        bool
	OnewayDefault bool
	isReversed    bool
}

func (wt *WayTags) IsPOI() bool {
	if wt.building != "" || wt.amenity != "" || wt.leisure != "" {
		return true
	}
	return false
}

func (way *WayTags) IsHighway() bool {
	return way.Highway != ""
}

func (wt *WayTags) IsHighwayPOI() bool {
	if _, ok := poiHighwayTags[wt.Highway]; ok {
		return true
	}
	return false
}

func (way *WayTags) IsRailway() bool {
	return way.Railway != ""
}

func (wt *WayTags) IsRailwayPOI() bool {
	if _, ok := poiRailwayTags[wt.Railway]; ok {
		return true
	}
	return false
}

func (way *WayTags) IsAeroway() bool {
	return way.Aeroway != ""
}

func (wt *WayTags) IsAerowayPOI() bool {
	if _, ok := poiAerowayTags[wt.Aeroway]; ok {
		return true
	}
	return false
}

func (wt *WayTags) IsHighwayNegligible() bool {
	_, ok := negligibleHighwayTags[wt.Highway]
	return ok
}

func NewWayTagsFrom(way *osm.Way) WayTags {
	tags := way.Tags

	name := tags.Find("name")
	highway := tags.Find("highway")
	railway := tags.Find("railway")
	aeroway := tags.Find("aeroway")

	turnLanes := tags.Find("turn:lanes")
	turnLanesForward := tags.Find("turn:lanes:forward")
	turnLanesBackward := tags.Find("turn:lanes:backward")

	area := tags.Find("area")
	motorVehicle := tags.Find("motor_vehicle")
	access := ""
	motorcar := tags.Find("motorcar")
	service := tags.Find("service")
	foot := tags.Find("foot")
	bicycle := tags.Find("bicycle")
	building := tags.Find("building")
	amenity := tags.Find("amenity")
	leisure := tags.Find("leisure")

	junction := tags.Find("junction")

	var err error

	maxSpeedSource := tags.Find("maxspeed")
	maxSpeed := -1.0
	if maxSpeedSource != "" {
		maxSpeedValue := -1.0
		kmhMaxSpeed := kmhRegExp.FindString(maxSpeedSource)
		if kmhMaxSpeed != "" {
			maxSpeedValue, err = strconv.ParseFloat(kmhMaxSpeed, 64)
			if err != nil {
				maxSpeedValue = -1
				log.Warn().Str("scope", "extract_way_tags").Any("osm_way_id", way.ID).Str("lanes:maxspeed", kmhMaxSpeed).Msg("Provided `lanes:maxspeed (km/h)` tag value should be an float (or integer?)")
			}
		} else {
			mphMaxSpeed := mphRegExp.FindString(maxSpeedSource)
			if mphMaxSpeed != "" {
				maxSpeedValue, err = strconv.ParseFloat(mphMaxSpeed, 64)
				if err != nil {
					maxSpeedValue = -1
					log.Warn().Str("scope", "extract_way_tags").Any("osm_way_id", way.ID).Str("lanes:maxspeed", mphMaxSpeed).Msg("Provided `lanes:maxspeed (mph)` tag value should be an float (or integer?)")
				}
			}
		}
		maxSpeed = maxSpeedValue
	}

	lanesSource := tags.Find("lanes")
	lanes := -1
	if lanesSource != "" {
		lanesNum := lanesRegExp.FindString(lanesSource)
		if lanesNum != "" {
			lanes, err = strconv.Atoi(lanesSource)
			if err != nil {
				lanes = -1
				log.Warn().Str("scope", "extract_way_tags").Any("osm_way_id", way.ID).Str("lanes", lanesSource).Msg("Provided `lanes` tag value should be an integer")
			}
		}
	}

	lanesForwardSource := tags.Find("lanes:forward")
	lanesForward := -1
	if lanesForwardSource != "" {
		lanesForward, err = strconv.Atoi(lanesForwardSource)
		if err != nil {
			lanesForward = -1
			log.Warn().Str("scope", "extract_way_tags").Any("osm_way_id", way.ID).Str("lanes:forward", lanesForwardSource).Msg("Provided `lanes:forward` tag value should be an integer")
		}
	}

	lanesBackwardSource := tags.Find("lanes:backward")
	lanesBackward := -1
	if lanesBackwardSource != "" {
		lanesBackward, err = strconv.Atoi(lanesBackwardSource)
		if err != nil {
			lanesBackward = -1
			log.Warn().Str("scope", "extract_way_tags").Any("osm_way_id", way.ID).Str("lanes:backward", lanesBackwardSource).Msg("Provided `lanes:backward` tag value should be an integer")
		}
	}

	oneway := false
	onewayDefault := false
	isReversed := false
	onewaySource := tags.Find("oneway")
	if onewaySource != "" {
		if onewaySource == "yes" || onewaySource == "1" {
			oneway = true
		} else if onewaySource == "no" || onewaySource == "0" {
			oneway = false
		} else if onewaySource == "-1" {
			oneway = true
			isReversed = true
		} else {
			// Reversible or alternating
			// Those are depends on time conditions
			// @TODO: need to implement
			if _, found := onewayReversible[onewaySource]; found {
				oneway = false
			} else {
				log.Warn().Str("scope", "extract_way_tags").Any("osm_way_id", way.ID).Str("oneway", onewaySource).Msg("Unhandled `oneway` tag value has been met")
			}
		}
	} else {
		if _, ok := junctionTypes[junction]; ok {
			oneway = true
		} else {
			oneway = false
			onewayDefault = true
		}
	}

	return WayTags{
		name:              name,
		Highway:           highway,
		Railway:           railway,
		Aeroway:           aeroway,
		turnLanes:         turnLanes,
		turnLanesForward:  turnLanesForward,
		turnLanesBackward: turnLanesBackward,
		junction:          junction,
		Area:              area,
		MotorVehicle:      motorVehicle,
		Access:            access,
		Motorcar:          motorcar,
		Service:           service,
		Foot:              foot,
		Bicycle:           bicycle,
		building:          building,
		amenity:           amenity,
		leisure:           leisure,
		maxSpeed:          maxSpeed,
		lanes:             lanes,
		lanesForward:      lanesForward,
		lanesBackward:     lanesBackward,
		Oneway:            oneway,
		OnewayDefault:     onewayDefault,
		isReversed:        isReversed,
	}
}
