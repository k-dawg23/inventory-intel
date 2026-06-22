CREATE TABLE IF NOT EXISTS ai_insight_runs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    actor TEXT NOT NULL,
    mode TEXT NOT NULL,
    status TEXT NOT NULL,
    model TEXT NOT NULL,
    window_days INTEGER NOT NULL,
    input_summary TEXT NOT NULL DEFAULT '',
    input_payload TEXT NOT NULL DEFAULT '',
    recommendations_json TEXT NOT NULL DEFAULT '[]',
    raw_output TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);
