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
		return invalid("metadata.channel must not be empty")
	}
	if e.Number < 1 {
		return invalid("metadata.messageNumber must be 1 or greater")
	}
	if e.Time.IsZero() {
		return invalid("metadata.messageTime is missing")
	}

	switch p := e.Payload.(type) {
	case models.Launched:
		if blank(p.RocketType) || blank(p.Mission) || p.LaunchSpeed < 0 {
			return invalid("launch needs a type, a mission and launchSpeed >= 0")
		}
	case models.SpeedIncreased:
		if p.By < 0 {
			return invalid("message.by must not be negative")
		}
	case models.SpeedDecreased:
		if p.By < 0 {
			return invalid("message.by must not be negative")
		}
	case models.Exploded:
		if blank(p.Reason) {
			return invalid("message.reason must not be empty")
		}
	case models.MissionChanged:
		if blank(p.NewMission) {
			return invalid("message.newMission must not be empty")
		}
	default:
		return invalid("unknown event payload")
	}
	return nil
}

func validateListOptions(o models.ListOptions) error {
	invalid := func(reason string) error { return fmt.Errorf("%w: %s", xerrors.ErrInvalidQuery, reason) }

	switch {
	case !sortFields[o.Sort]:
		return invalid("sort accepts channel, type, speed or mission")
	case o.Order != "asc" && o.Order != "desc":
		return invalid("order accepts asc or desc")
	case o.Limit < 1 || o.Limit > 1000:
		return invalid("limit accepts 1 to 1000")
	case o.Offset < 0:
		return invalid("offset must not be negative")
	}
	return nil
}
