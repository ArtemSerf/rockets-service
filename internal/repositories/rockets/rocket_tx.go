package rockets

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"

	"rockets/internal/models"
)

type Tx struct {
	tx      pgx.Tx
	channel string
}

func (t *Tx) InsertEvent(ctx context.Context, e models.Event, receivedAt time.Time) (bool, error) {
	payload, err := json.Marshal(e.Payload)
	if err != nil {
		return false, err
	}
	tag, err := t.tx.Exec(ctx, `
		INSERT INTO rocket_events (channel, message_number, message_type, message_time, payload, received_at)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6)
		ON CONFLICT DO NOTHING`,
		e.Channel, e.Number, string(e.Payload.Type()), e.Time, string(payload), receivedAt)
	return tag.RowsAffected() == 1, err
}

func (t *Tx) PendingEvents(ctx context.Context, after int64, limit int) ([]models.Event, error) {
	rows, err := t.tx.Query(ctx, `
		SELECT message_number, message_type, message_time, payload
		FROM rocket_events
		WHERE channel = $1 AND message_number > $2 AND message_number <= $2 + $3 AND status = 'pending'
		ORDER BY message_number`, t.channel, after, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.Event
	for rows.Next() {
		e := models.Event{Channel: t.channel}
		var kind string
		var payload []byte
		if err := rows.Scan(&e.Number, &kind, &e.Time, &payload); err != nil {
			return nil, err
		}
		if e.Payload, err = models.DecodePayload(models.EventType(kind), payload); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (t *Tx) MarkEvents(ctx context.Context, results []models.EventResult) error {
	numbers := make([]int64, len(results))
	statuses := make([]string, len(results))
	reasons := make([]string, len(results))
	for i, r := range results {
		numbers[i], statuses[i], reasons[i] = r.Number, string(r.Status), r.Reason
	}

	_, err := t.tx.Exec(ctx, `
		UPDATE rocket_events e SET status = u.status, reject_reason = NULLIF(u.reason, '')
		FROM unnest($2::bigint[], $3::text[], $4::text[]) AS u(number, status, reason)
		WHERE e.channel = $1 AND e.message_number BETWEEN $5 AND $6 AND e.message_number = u.number`,
		t.channel, numbers, statuses, reasons, numbers[0], numbers[len(numbers)-1])
	return err
}

func (t *Tx) SaveRocket(ctx context.Context, r models.Rocket) error {
	_, err := t.tx.Exec(ctx, `
		UPDATE rockets SET status = $2, rocket_type = $3, speed = $4, mission = $5, explosion_reason = $6,
			last_processed_number = $7, last_message_time = $8, updated_at = now()
		WHERE channel = $1`,
		t.channel, string(r.Status), r.Type, r.Speed, r.Mission, r.ExplosionReason,
		r.LastProcessed, r.LastMessageTime)
	return err
}
