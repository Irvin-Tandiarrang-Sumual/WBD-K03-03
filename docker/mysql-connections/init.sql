-- Case 3: a MySQL analytics database for testing the second database driver.
CREATE TABLE analytics_events (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    event_name VARCHAR(80) NOT NULL,
    source VARCHAR(40) NOT NULL,
    occurred_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE daily_metrics (
    metric_date DATE PRIMARY KEY,
    page_views INT UNSIGNED NOT NULL,
    unique_visitors INT UNSIGNED NOT NULL
);

INSERT INTO analytics_events (event_name, source) VALUES
    ('page_view', 'web'),
    ('checkout_completed', 'web');

INSERT INTO daily_metrics (metric_date, page_views, unique_visitors) VALUES
    ('2026-10-24', 1280, 640),
    ('2026-10-25', 1510, 702);
