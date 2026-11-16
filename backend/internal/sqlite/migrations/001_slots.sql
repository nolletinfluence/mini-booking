CREATE TABLE slots (
    id INTEGER PRIMARY KEY,
    starts_at INTEGER NOT NULL UNIQUE,
    ends_at INTEGER NOT NULL CHECK (ends_at > starts_at)
);
