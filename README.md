# Rockets service

A Go service that consumes rocket messages and exposes the current state of each rocket over a REST API.
Messages may arrive out of order and more than once; the service applies them strictly in
`messageNumber` order per channel, exactly once. State is stored in PostgreSQL.

Design decisions and trade-offs: [DESIGN.md](DESIGN.md).

## Run

Requirements: Docker with Compose. Port `8088` must be free.

```sh
make up                                   # postgres + migrations + service
curl http://localhost:8088/health         # {"status":"ok"}
```

Then start the test program (pick the binary for your platform):

```sh
./rockets launch "http://localhost:8088/messages" --message-delay=500ms --concurrency-level=1
```

`make down` stops the stack and keeps data, `make reset` also deletes the database.

## API

| Endpoint | Description |
| --- | --- |
| `POST /messages` | Ingest one message |
| `GET /rockets/{channel}` | State of one rocket |
| `GET /rockets` | List of launched rockets, sortable |
| `GET /health` | 200 if the database is reachable |

### `POST /messages`

| Code | Meaning |
| --- | --- |
| 200 `{"status":"accepted"}` | Stored. Applied, waiting behind a gap, or rejected by business rules |
| 200 `{"status":"duplicate"}` | Already received; nothing changed |
| 400 | Malformed message (bad JSON, unknown type, missing fields) |
| 503 | Not stored (database error or timeout); the producer should retry |

A 200 is only returned after the transaction commits.

### `GET /rockets/{channel}`

```json
{
  "channel": "193270a9-c9cf-404a-8f83-838e71d9ae67",
  "status": "exploded",
  "type": "Falcon-9",
  "speed": 3000,
  "mission": "ARTEMIS",
  "explosionReason": "PRESSURE_VESSEL_FAILURE",
  "lastProcessedMessageNumber": 5,
  "lastMessageTime": "2022-02-02T18:39:05.5Z",
  "pendingMessages": 0,
  "hasGap": false
}
```

- `status`: `pending` (messages received, launch not processed yet), `flying` or `exploded`.
- `pendingMessages` / `hasGap`: messages stored behind a missing one. `hasGap: false` does not
  guarantee the state is final; more messages may still be in transit.
- 404 if no message for the channel has ever been received.

### `GET /rockets`

Returns `{"rockets": [...], "limit": 100, "offset": 0}`. Only launched rockets are listed.

| Parameter | Default | Values |
| --- | --- | --- |
| `sort` | `channel` | `channel`, `type`, `speed`, `mission` |
| `order` | `asc` | `asc`, `desc` |
| `limit` | `100` | 1..1000 |
| `offset` | `0` | ≥ 0 |

Ties are broken by channel. Unknown or repeated parameters return 400.

## Development

```sh
make test     # service tests against Postgres from compose
make logs     # follow service logs
make run BIN=./rockets DURATION=60   # start the stack, run the test program for 60s, print the top rockets
```

Running the service outside Docker:

```sh
docker compose up -d postgres migrate
DATABASE_URL='postgres://rockets:rockets@localhost:5433/rockets?sslmode=disable' go run ./cmd/rockets
```

## Layout

```
cmd/rockets                    wiring, HTTP server, graceful shutdown
internal/controllers/http      HTTP handlers, request/response conversion
internal/services/rockets      ingestion (ordering, idempotency), validation, rocket rules
internal/repositories/rockets  PostgreSQL: row lock, events, snapshots, reads
internal/models                events and rockets
internal/xerrors               error kinds mapped to HTTP codes
internal/libs/log              structured logging
migrations                     SQL schema (golang-migrate)
```
