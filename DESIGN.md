# Design notes

## The problem

Every rocket reports changes on its own channel. Messages come out of order, can arrive more than
once, and are resent whenever the service answers with anything other than 2xx. From that stream
I need to keep the current state of each rocket and serve it to a dashboard.

## Assumptions

- `channel` is the rocket ID. `messageNumber` defines the order inside a channel: it starts at 1
  with `RocketLaunched` and has no holes. `messageTime` is only shown, never used for ordering.
- A message is identified by `(channel, messageNumber)`. I assume the producer always sends the
  same content for the same key; if it doesn't, the first version wins.
- The task leaves some rules open, so I picked these:
  - speed never drops below zero; such a decrease is clamped and logged;
  - an exploded rocket is final, so anything after the explosion is rejected;
  - a second launch, or anything before the launch, is rejected.

## Overall shape

One Go service and PostgreSQL.

```
POST /messages -> controllers/http -> services/rockets (Ingest) -> repositories/rockets
GET  /rockets  -> controllers/http -> services/rockets          -> repositories/rockets
```

The rocket rules live in one pure function, `apply(rocket, payload)`, in
`services/rockets/apply.go`. It knows nothing about ordering, HTTP or the database. Ordering and
deduplication are handled in `services/rockets/ingest.go`.

I didn't put interfaces between the service and the repository. There is only one storage, and
the parts that can actually break (locking, `ON CONFLICT`) are SQL, so they need to be tested
against a real database anyway.

I didn't use Kafka or another broker. A broker keeps order within a partition, but here messages
are already out of order when they reach us, so I'd still need ordering in the application.
Postgres gives durable storage and atomic updates with a single dependency.

## Data

- `rocket_events` keeps every unique message. Its primary key `(channel, message_number)` is the
  idempotency key. Each event is `pending` (waiting for an earlier one), `applied`, or `rejected`
  with a reason.
- `rockets` keeps the current state and `last_processed_number`: everything up to that number has
  been handled. The rocket has a single `status` (`pending`, `flying`, `exploded`) rather than two
  booleans, so a state like "exploded but never launched" can't exist.

Keeping the full history costs disk space, but it lets me rebuild state if the rules change and
see exactly what happened to any rocket.

## Ingesting a message

Saving a message and applying it are two separate steps.

1. **Save.** In a short transaction: lock the rocket, insert the event with
   `ON CONFLICT DO NOTHING` (no row inserted means duplicate), commit. The event is now safe.
2. **Apply.** In batches of up to 500, each in its own transaction: lock the rocket, take the
   pending events right after `last_processed_number`, apply them in order until the first gap,
   update their statuses with one query, save the rocket, commit. Repeat while batches are full.
3. Answer 200. If applying fails or runs out of time, it is logged; finished batches stay
   committed and the next message for that rocket picks up where it stopped.

The lock is one statement: `INSERT ... ON CONFLICT (channel) DO UPDATE ... RETURNING ...`. It
creates the rocket if needed, locks it until commit and returns its state in one round trip.

**Why two steps.** If saving and applying shared one transaction, a long gap would break the
rocket: the message closing it would have to apply the whole backlog, could hit the request
timeout, get rolled back and resent, and fail the same way again. With the event saved first, a
slow apply never causes a retry, and progress is kept batch by batch.

**Concurrency.** The row lock means one rocket is processed by one request at a time, even with
several service instances. Different rockets run in parallel. I didn't add an in-memory mutex: it
would only work inside one process and the database already guarantees this. Small batches keep
each lock short.

**Response codes.** The producer retries anything that isn't 2xx, so each code is a decision about
retries:

- **200** when the event is saved: applied, waiting behind a gap, or rejected by the rules.
  Answering an error to a valid but conflicting event would make the producer resend it forever
  and block the rocket.
- **400** for malformed messages. A retry won't fix them, so nothing is saved.
- **503** when the event couldn't be saved. Here a retry is exactly what I want.

## Gaps

If a message never arrives, the rocket stays at its last consistent state. I don't skip gaps
automatically, because that would quietly produce a wrong state. Gaps are visible instead:
`pendingMessages` and `hasGap` in the API, and `pending` rows in the database.

## Reading

Reads don't take locks and never create rows. A rocket and its pending count come from one query,
so they always match. The list is sorted in SQL with an allowlist of columns and channel as the
tie-breaker, and uses offset pagination. Rockets can move between pages while they update, which
is fine for a dashboard.
