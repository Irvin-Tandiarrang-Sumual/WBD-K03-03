CREATE TABLE customers (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(120) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE orders (
    id BIGSERIAL PRIMARY KEY,
    customer_id BIGINT NOT NULL REFERENCES customers(id),
    total_amount NUMERIC(12, 2) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE order_items (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES orders(id),
    item_name VARCHAR(120) NOT NULL,
    quantity INT NOT NULL DEFAULT 1,
    price NUMERIC(12, 2) NOT NULL
);

INSERT INTO customers (name, email) VALUES
    ('Rina Pratama', 'rina@example.com'),
    ('Budi Santoso', 'budi@example.com'),
    ('Siti Aminah', 'siti@example.com'),
    ('Dewi Lestari', 'dewi@example.com'),
    ('Agus Setiawan', 'agus@example.com'),
    ('Fajar Nugraha', 'fajar@example.com'),
    ('Indah Permata', 'indah@example.com'),
    ('Rizky Ramadhan', 'rizky@example.com'),
    ('Maya Anggraini', 'maya@example.com'),
    ('Hendra Wijaya', 'hendra@example.com'),
    ('Fitri Handayani', 'fitri@example.com'),
    ('Bayu Pratama', 'bayu@example.com'),
    ('Lestari Ningsih', 'lestari@example.com'),
    ('Bambang Pamungkas', 'bambang@example.com'),
    ('Anisa Rahma', 'anisa@example.com'),
    ('Dimas Anggara', 'dimas@example.com'),
    ('Eko Prasetyo', 'eko@example.com'),
    ('Gita Gutawa', 'gita@example.com'),
    ('Surya Saputra', 'surya@example.com'),
    ('Tari Wulandari', 'tari@example.com');

INSERT INTO orders (customer_id, total_amount, status) VALUES
    (1, 1250000.00, 'paid'),
    (1, 450000.00, 'completed'),
    (2, 750000.00, 'pending'),
    (3, 2100000.00, 'paid'),
    (4, 320000.00, 'cancelled'),
    (5, 890000.00, 'completed'),
    (6, 1500000.00, 'paid'),
    (7, 670000.00, 'pending'),
    (8, 2300000.00, 'completed'),
    (9, 450000.00, 'paid'),
    (10, 1100000.00, 'completed'),
    (11, 780000.00, 'paid'),
    (12, 950000.00, 'pending'),
    (13, 1400000.00, 'completed'),
    (14, 520000.00, 'paid'),
    (15, 3100000.00, 'paid'),
    (16, 680000.00, 'pending'),
    (17, 890000.00, 'completed'),
    (18, 1750000.00, 'paid'),
    (19, 4200000.00, 'completed'),
    (20, 2500000.00, 'paid');

INSERT INTO order_items (order_id, item_name, quantity, price) VALUES
    (1, 'Logitech MX Master 3S', 1, 1250000.00),
    (2, 'Keychron K2 Wireless Keyboard', 1, 450000.00),
    (3, 'Sony WH-1000XM5 Headphones', 1, 750000.00),
    (4, 'Dell UltraSharp 27 Monitor', 1, 2100000.00),
    (5, 'Anker USB-C Fast Charger', 2, 160000.00),
    (6, 'Apple Magic Trackpad', 1, 890000.00),
    (7, 'Samsung 990 Pro 2TB NVMe', 1, 1500000.00),
    (8, 'Ergonomic Desk Mat Leather', 2, 335000.00),
    (9, 'CalDigit TS4 Thunderbolt Dock', 1, 2300000.00),
    (10, 'BenQ ScreenBar Halo', 1, 450000.00),
    (11, 'Secretlab Titan Evo Chair', 1, 1100000.00),
    (12, 'Logitech Brio 4K Webcam', 1, 780000.00),
    (13, 'Elgato Stream Deck MK.2', 1, 950000.00),
    (14, 'Shure SM7B Microphone', 1, 1400000.00),
    (15, 'Focusrite Scarlett 2i2 Interface', 1, 520000.00),
    (16, 'LG OLED 42 C3 Display', 1, 3100000.00),
    (17, 'Audioengine A2+ Wireless Speakers', 1, 680000.00),
    (18, 'Steelcase Gesture Headrest', 1, 890000.00),
    (19, 'Bose QuietComfort Ultra', 1, 1750000.00),
    (20, 'Herman Miller Embody Gaming Chair', 1, 4200000.00),
    (21, 'Razer DeathAdder V3 Pro', 2, 1250000.00);

GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO demo_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO demo_user;
