package rockets

import (
	"fmt"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"rockets/internal/models"
	repository "rockets/internal/repositories/rockets"
	"rockets/internal/xerrors"
)

type fixture struct {
	t       *testing.T
	service *Service
	channel string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set, run: make test")
	}
	repo, err := repository.NewRepository(t.Context(), dsn)
	require.NoError(t, err)
	t.Cleanup(repo.Close)

	return &fixture{
		t:       t,
		service: NewService(repo),
		channel: fmt.Sprintf("%s-%d", t.Name(), time.Now().UnixNano()),
	}
}

func (f *fixture) event(number int64, payload models.Payload) models.Event {
	return models.Event{
		Channel: f.channel,
		Number:  number,
		Time:    time.Date(2022, 2, 2, 19, 0, 0, 0, time.UTC).Add(time.Duration(number) * time.Second),
		Payload: payload,
	}
}

func (f *fixture) send(events ...models.Event) {
	f.t.Helper()
	for _, e := range events {
		_, err := f.service.Ingest(f.t.Context(), e)
		require.NoError(f.t, err)
	}
}

func (f *fixture) rocket() models.Rocket {
	f.t.Helper()
	rocket, err := f.service.Rocket(f.t.Context(), f.channel)
	require.NoError(f.t, err)
	return rocket
}

func (f *fixture) flight() []models.Event {
	return []models.Event{
		f.event(1, models.Launched{RocketType: "Falcon-9", LaunchSpeed: 500, Mission: "ARTEMIS"}),
		f.event(2, models.SpeedIncreased{By: 3000}),
		f.event(3, models.MissionChanged{NewMission: "SHUTTLE_MIR"}),
		f.event(4, models.SpeedDecreased{By: 2500}),
		f.event(5, models.SpeedIncreased{By: 700}),
		f.event(6, models.Exploded{Reason: "PRESSURE_VESSEL_FAILURE"}),
		f.event(7, models.SpeedIncreased{By: 100}),
	}
}

func (f *fixture) requireFlightFinished() {
	f.t.Helper()
	rocket := f.rocket()
	require.Equal(f.t, models.RocketExploded, rocket.Status)
	require.Equal(f.t, "Falcon-9", rocket.Type)
	require.Equal(f.t, int64(1700), rocket.Speed)
	require.Equal(f.t, "SHUTTLE_MIR", rocket.Mission)
	require.Equal(f.t, "PRESSURE_VESSEL_FAILURE", rocket.ExplosionReason)
	require.Equal(f.t, int64(7), rocket.LastProcessed)
	require.Zero(f.t, rocket.PendingCount)
}

func TestService_Ingest(t *testing.T) {
	t.Run("events in order are applied", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)

		f.send(f.flight()...)

		f.requireFlightFinished()
	})

	t.Run("events in reverse order are applied once the first one arrives", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		events := f.flight()

		for i := len(events) - 1; i >= 0; i-- {
			f.send(events[i])
		}

		f.requireFlightFinished()
	})

	t.Run("events after a gap wait until the gap is filled", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		events := f.flight()

		f.send(events[0], events[2], events[3])

		rocket := f.rocket()
		require.Equal(t, int64(1), rocket.LastProcessed)
		require.Equal(t, int64(500), rocket.Speed)
		require.Equal(t, int64(2), rocket.PendingCount)

		f.send(events[1])

		rocket = f.rocket()
		require.Equal(t, int64(4), rocket.LastProcessed)
		require.Equal(t, int64(1000), rocket.Speed)
		require.Equal(t, "SHUTTLE_MIR", rocket.Mission)
		require.Zero(t, rocket.PendingCount)
	})

	t.Run("duplicate is reported and changes nothing", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		events := f.flight()
		f.send(events[0], events[1])
		before := f.rocket()

		duplicate, err := f.service.Ingest(t.Context(), events[1])
		require.NoError(t, err)
		require.True(t, duplicate)

		duplicate, err = f.service.Ingest(t.Context(), f.event(2, models.SpeedIncreased{By: 999999}))
		require.NoError(t, err)
		require.True(t, duplicate)

		require.Equal(t, before, f.rocket())
	})

	t.Run("rejected event does not block the sequence", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)

		f.send(
			f.event(1, models.Launched{RocketType: "X", LaunchSpeed: 10, Mission: "M"}),
			f.event(2, models.Launched{RocketType: "Y", LaunchSpeed: 99, Mission: "N"}),
			f.event(3, models.SpeedIncreased{By: 5}),
		)

		rocket := f.rocket()
		require.Equal(t, "X", rocket.Type)
		require.Equal(t, int64(15), rocket.Speed)
		require.Equal(t, int64(3), rocket.LastProcessed)
	})

	t.Run("speed does not go below zero", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)

		f.send(
			f.event(1, models.Launched{RocketType: "X", LaunchSpeed: 100, Mission: "M"}),
			f.event(2, models.SpeedDecreased{By: 300}),
			f.event(3, models.SpeedIncreased{By: 50}),
		)

		require.Equal(t, int64(50), f.rocket().Speed)
	})

	t.Run("invalid event is not stored", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)

		_, err := f.service.Ingest(t.Context(), f.event(1, models.Exploded{}))
		require.ErrorIs(t, err, xerrors.ErrInvalidEvent)

		_, err = f.service.Rocket(t.Context(), f.channel)
		require.ErrorIs(t, err, xerrors.ErrNotFound)
	})

	t.Run("any delivery order with duplicates gives the same state", func(t *testing.T) {
		t.Parallel()
		for seed := range int64(20) {
			f := newFixture(t)
			f.channel = fmt.Sprintf("%s-seed-%d", f.channel, seed)
			rng := rand.New(rand.NewSource(seed))
			events := f.flight()
			delivery := append(events, events[rng.Intn(len(events))], events[rng.Intn(len(events))])
			rng.Shuffle(len(delivery), func(i, j int) { delivery[i], delivery[j] = delivery[j], delivery[i] })

			f.send(delivery...)

			f.requireFlightFinished()
		}
	})

	t.Run("concurrent duplicates are applied once", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)

		var wg sync.WaitGroup
		for _, e := range f.flight() {
			for range 3 {
				wg.Go(func() {
					_, err := f.service.Ingest(t.Context(), e)
					require.NoError(t, err)
				})
			}
		}
		wg.Wait()

		f.requireFlightFinished()
	})

	t.Run("backlog larger than one batch is applied in full", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		total := int64(batchSize*3 + 7)

		f.send(f.event(1, models.Launched{RocketType: "X", LaunchSpeed: 0, Mission: "M"}))
		for n := int64(3); n <= total; n++ {
			f.send(f.event(n, models.SpeedIncreased{By: 1}))
		}
		require.Equal(t, int64(1), f.rocket().LastProcessed)

		f.send(f.event(2, models.SpeedIncreased{By: 1}))

		rocket := f.rocket()
		require.Equal(t, total, rocket.LastProcessed)
		require.Equal(t, total-1, rocket.Speed)
		require.Zero(t, rocket.PendingCount)
	})
}
