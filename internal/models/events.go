package models

import (
	"encoding/json"
	"fmt"
	"time"
)

type EventType string

const (
	TypeLaunched       EventType = "RocketLaunched"
	TypeSpeedIncreased EventType = "RocketSpeedIncreased"
	TypeSpeedDecreased EventType = "RocketSpeedDecreased"
	TypeExploded       EventType = "RocketExploded"
	TypeMissionChanged EventType = "RocketMissionChanged"
)

type EventStatus string

const (
	EventPending  EventStatus = "pending"
	EventApplied  EventStatus = "applied"
	EventRejected EventStatus = "rejected"
)

type EventResult struct {
	Number int64
	Status EventStatus
	Reason string
}

type Event struct {
	Channel string
	Number  int64
	Time    time.Time
	Payload Payload
}

type Payload interface {
	Type() EventType
}

type Launched struct {
	RocketType  string `json:"type"`
	LaunchSpeed int64  `json:"launchSpeed"`
	Mission     string `json:"mission"`
}

type SpeedIncreased struct {
	By int64 `json:"by"`
}

type SpeedDecreased struct {
	By int64 `json:"by"`
}

type Exploded struct {
	Reason string `json:"reason"`
}

type MissionChanged struct {
	NewMission string `json:"newMission"`
}

func (Launched) Type() EventType       { return TypeLaunched }
func (SpeedIncreased) Type() EventType { return TypeSpeedIncreased }
func (SpeedDecreased) Type() EventType { return TypeSpeedDecreased }
func (Exploded) Type() EventType       { return TypeExploded }
func (MissionChanged) Type() EventType { return TypeMissionChanged }

var decoders = map[EventType]func([]byte) (Payload, error){
	TypeLaunched:       decode[Launched],
	TypeSpeedIncreased: decode[SpeedIncreased],
	TypeSpeedDecreased: decode[SpeedDecreased],
	TypeExploded:       decode[Exploded],
	TypeMissionChanged: decode[MissionChanged],
}

func DecodePayload(t EventType, data []byte) (Payload, error) {
	decoder, ok := decoders[t]
	if !ok {
		return nil, fmt.Errorf("unknown event type %q", t)
	}
	return decoder(data)
}

func decode[T Payload](data []byte) (Payload, error) {
	var p T
	err := json.Unmarshal(data, &p)
	return p, err
}
