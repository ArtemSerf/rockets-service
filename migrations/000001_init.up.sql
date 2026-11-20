CREATE TABLE rockets (
    channel               text   PRIMARY KEY CHECK (channel <> ''),
    status                text   NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'flying', 'exploded')),
    rocket_type           text   NOT NULL DEFAULT '',
    speed                 bigint NOT NULL DEFAULT 0 CHECK (speed >= 0),
    mission               text   NOT NULL DEFAULT '',
    explosion_reason      text   NOT NULL DEFAULT '',
    last_processed_number bigint NOT NULL DEFAULT 0 CHECK (last_processed_number >= 0),
    last_message_time     timestamptz,
    updated_at            timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE rocket_events (
    channel        text        NOT NULL REFERENCES rockets (channel),
    message_number bigint      NOT NULL CHECK (message_number > 0),
    message_type   text        NOT NULL,
    message_time   timestamptz NOT NULL,
    payload        jsonb       NOT NULL,
    received_at    timestamptz NOT NULL,
    status         text        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'applied', 'rejected')),
    reject_reason  text,
    PRIMARY KEY (channel, message_number)
);

CREATE INDEX rocket_events_pending_idx ON rocket_events (channel, message_number) WHERE status = 'pending';
