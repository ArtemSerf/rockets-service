package httpapi

import (
	"errors"
	"fmt"
	"time"

	"rockets/internal/models"
)

type message struct {
	Metadata struct {
		Channel       string           `json:"channel"`
		MessageNumber int64            `json:"messageNumber"`
		MessageTime   time.Time        `json:"messageTime"`
		MessageType   models.EventType `json:"messageType"`
	} `json:"metadata"`
	Message struct {
		Type        string `json:"type"`
		LaunchSpeed *int64 `json:"launchSpeed"`
		Mission     string `json:"mission"`
		By          *int64 `json:"by"`
		Reason      string `json:"reason"`
		NewMission  string `json:"newMission"`
	} `json:"message"`
}

func (m message) toEvent() (models.Event, error) {
	body := m.Message
	var payload models.Payload
	switch m.Metadata.MessageType {
	case models.TypeLaunched:
		if body.LaunchSpeed == nil {
			return models.Event{}, errors.New("launchSpeed is required")
		}
		payload = models.Launched{RocketType: body.Type, LaunchSpeed: *body.LaunchSpeed, Mission: body.Mission}
	case models.TypeSpeedIncreased:
		if body.By == nil {
			return models.Event{}, errors.New("by is required")
		}
		payload = models.SpeedIncreased{By: *body.By}
	case models.TypeSpeedDecreased:
		if body.By == nil {
			return models.Event{}, errors.New("by is required")
		}
		payload = models.SpeedDecreased{By: *body.By}
	case models.TypeExploded:
		payload = models.Exploded{Reason: body.Reason}
	case models.TypeMissionChanged:
		payload = models.MissionChanged{NewMission: body.NewMission}
	default:
		return models.Event{}, fmt.Errorf("unknown messageType %q", m.Metadata.MessageType)
	}

	return models.Event{
		Channel: m.Metadata.Channel,
		Number:  m.Metadata.MessageNumber,
		Time:    m.Metadata.MessageTime,
		Payload: payload,
	}, nil
}

type rocketResponse struct {
	Channel         string     `json:"channel"`
	Status          string     `json:"status"`
	Type            string     `json:"type,omitempty"`
	Speed           int64      `json:"speed"`
	Mission         string     `json:"mission,omitempty"`
	ExplosionReason string     `json:"explosionReason,omitempty"`
	LastMessage     int64      `json:"lastProcessedMessageNumber"`
	LastMessageTime *time.Time `json:"lastMessageTime,omitempty"`
	PendingMessages int64      `json:"pendingMessages"`
	HasGap          bool       `json:"hasGap"`
}

func toRocketResponse(r models.Rocket) rocketResponse {
	resp := rocketResponse{
		Channel:         r.Channel,
		Status:          string(r.Status),
		Type:            r.Type,
		Speed:           r.Speed,
		Mission:         r.Mission,
		ExplosionReason: r.ExplosionReason,
		LastMessage:     r.LastProcessed,
		PendingMessages: r.PendingCount,
		HasGap:          r.PendingCount > 0,
	}
	if !r.LastMessageTime.IsZero() {
		resp.LastMessageTime = &r.LastMessageTime
	}
	return resp
}
