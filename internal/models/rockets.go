package models

import "time"

type RocketStatus string

const (
	RocketPending  RocketStatus = "pending"
	RocketFlying   RocketStatus = "flying"
	RocketExploded RocketStatus = "exploded"
)

type Rocket struct {
	Channel         string
	Status          RocketStatus
	Type            string
	Speed           int64
	Mission         string
	ExplosionReason string
	LastProcessed   int64
	LastMessageTime time.Time
	PendingCount    int64
}

type ListOptions struct {
	Sort   string
	Order  string
	Limit  int
	Offset int
}

func DefaultListOptions() ListOptions {
	return ListOptions{Sort: "channel", Order: "asc", Limit: 100}
}
