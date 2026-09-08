CREATE TABLE IF NOT EXISTS metrics (
    -- The name comes from the client, so the bound is generous.
    id    varchar(255) PRIMARY KEY,
    -- Only 'gauge' and 'counter' ever get here.
    type  varchar(16) NOT NULL,
    delta bigint,
    value double precision
);
