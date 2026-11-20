package rockets

import (
	"context"
	"fmt"

	"rockets/internal/libs/log"
	"rockets/internal/models"
	repository "rockets/internal/repositories/rockets"
)

const batchSize = 500

func (s *Service) Ingest(ctx context.Context, e models.Event) (duplicate bool, err error) {
	if err := validateEvent(e); err != nil {
		return false, err
	}

	err = s.repository.WithRocketLock(ctx, e.Channel, func(tx *repository.Tx, _ models.Rocket) error {
		inserted, err := tx.InsertEvent(ctx, e, s.now().UTC())
		duplicate = !inserted
		return err
	})
	if err != nil {
		return false, fmt.Errorf("store %s #%d: %w", e.Channel, e.Number, err)
	}

	if err := s.processPending(ctx, e.Channel); err != nil {
		log.Error(ctx, err, "process pending events", "channel", e.Channel)
	}
	return duplicate, nil
}

func (s *Service) processPending(ctx context.Context, channel string) error {
	for {
		processed := 0
		err := s.repository.WithRocketLock(ctx, channel, func(tx *repository.Tx, rocket models.Rocket) error {
			events, err := tx.PendingEvents(ctx, rocket.LastProcessed, batchSize)
			if err != nil {
				return err
			}

			results := make([]models.EventResult, 0, len(events))
			for _, e := range events {
				if e.Number != rocket.LastProcessed+1 {
					break
				}

				result := models.EventResult{Number: e.Number, Status: models.EventApplied}
				next, err := apply(ctx, rocket, e.Payload)
				if err != nil {
					result.Status, result.Reason = models.EventRejected, err.Error()
					log.Info(ctx, "event rejected", "channel", channel, "message_number", e.Number, "reason", result.Reason)
				} else {
					rocket = next
				}
				rocket.LastProcessed, rocket.LastMessageTime = e.Number, e.Time
				results = append(results, result)
			}

			processed = len(results)
			if processed == 0 {
				return nil
			}
			if err := tx.MarkEvents(ctx, results); err != nil {
				return err
			}
			return tx.SaveRocket(ctx, rocket)
		})
		if err != nil {
			return err
		}
		if processed < batchSize {
			return nil
		}
	}
}
