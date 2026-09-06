CREATE TABLE IF NOT EXISTS metrics (
    id    text PRIMARY KEY,
    type  text NOT NULL,
    delta bigint,
    value double precision
);
