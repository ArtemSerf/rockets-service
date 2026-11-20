package rockets

import (
	"context"
	"errors"
	"math"

	"rockets/internal/libs/log"
	"rockets/internal/models"
)

func apply(ctx context.Context, r models.Rocket, p models.Payload) (models.Rocket, error) {
	if r.Status == models.RocketExploded {
		return r, errors.New("rocket already exploded")
	}
	if _, isLaunch := p.(models.Launched); r.Status == models.RocketPending && !isLaunch {
		return r, errors.New("rocket not launched")
	}

	switch p := p.(type) {
	case models.Launched:
		if r.Status != models.RocketPending {
			return r, errors.New("rocket already launched")
		}
		r.Status, r.Type, r.Speed, r.Mission = models.RocketFlying, p.RocketType, p.LaunchSpeed, p.Mission
	case models.SpeedIncreased:
		if r.Speed > math.MaxInt64-p.By {
			return r, errors.New("speed overflow")
		}
		r.Speed += p.By
	case models.SpeedDecreased:
		if p.By > r.Speed {
			log.Info(ctx, "speed clamped to zero", "channel", r.Channel, "speed", r.Speed, "decrease", p.By)
		}
		r.Speed = max(r.Speed-p.By, 0)
	case models.MissionChanged:
		r.Mission = p.NewMission
	case models.Exploded:
		r.Status, r.ExplosionReason = models.RocketExploded, p.Reason
	default:
		return r, errors.New("unsupported event")
	}
	return r, nil
}
