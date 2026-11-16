CREATE TABLE bookings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    slot_id INTEGER NOT NULL UNIQUE REFERENCES slots(id),
    customer_name TEXT NOT NULL,
    customer_email TEXT NOT NULL
);
