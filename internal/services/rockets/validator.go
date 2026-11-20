package rockets

import (
	"fmt"
	"strings"

	"rockets/internal/models"
	"rockets/internal/xerrors"
)

var sortFields = map[string]bool{"channel": true, "type": true, "speed": true, "mission": true}

func validateEvent(e models.Event) error {
	invalid := func(reason string) error { return fmt.Errorf("%w: %s", xerrors.ErrInvalidEvent, reason) }
	blank := func(s string) bool { return strings.TrimSpace(s) == "" }

	if blank(e.Channel) {
		return invalid("channel is required")
	}
	if e.Number < 1 {
		return invalid("message number must be positive")
	}
	if e.Time.IsZero() {
		return invalid("message time is required")
	}

	switch p := e.Payload.(type) {
	case models.Launched:
		if blank(p.RocketType) || blank(p.Mission) || p.LaunchSpeed < 0 {
			return invalid("launch requires type, mission and non-negative speed")
		}
	case models.SpeedIncreased:
		if p.By < 0 {
			return invalid("speed change must be non-negative")
		}
	case models.SpeedDecreased:
		if p.By < 0 {
			return invalid("speed change must be non-negative")
		}
	case models.Exploded:
		if blank(p.Reason) {
			return invalid("explosion reason is required")
		}
	case models.MissionChanged:
		if blank(p.NewMission) {
			return invalid("new mission is required")
		}
	default:
		return invalid("unsupported payload")
	}
	return nil
}

func validateListOptions(o models.ListOptions) error {
	invalid := func(reason string) error { return fmt.Errorf("%w: %s", xerrors.ErrInvalidQuery, reason) }

	switch {
	case !sortFields[o.Sort]:
		return invalid("sort must be one of channel, type, speed, mission")
	case o.Order != "asc" && o.Order != "desc":
		return invalid("order must be asc or desc")
	case o.Limit < 1 || o.Limit > 1000:
		return invalid("limit must be between 1 and 1000")
	case o.Offset < 0:
		return invalid("offset must be non-negative")
	}
	return nil
}
