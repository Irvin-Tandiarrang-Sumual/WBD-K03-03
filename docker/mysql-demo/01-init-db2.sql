CREATE TABLE products (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    sku VARCHAR(40) NOT NULL UNIQUE,
    name VARCHAR(120) NOT NULL,
    price DECIMAL(12, 2) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE inventory_movements (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    product_id BIGINT UNSIGNED NOT NULL,
    quantity_change INT NOT NULL,
    movement_type VARCHAR(24) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT inventory_product_fk
        FOREIGN KEY (product_id) REFERENCES products(id)
);

INSERT INTO products (sku, name, price) VALUES
    ('WBD-001', 'Keychron Q1 Pro Custom Mechanical Keyboard', 2850000.00),
    ('WBD-002', 'Logitech MX Master 3S Ergonomic Mouse', 1350000.00),
    ('WBD-003', 'CalDigit TS4 Thunderbolt 4 Station Dock', 5800000.00),
    ('WBD-004', 'Dell UltraSharp 32-inch 4K USB-C Hub Monitor', 9200000.00),
    ('WBD-005', 'BenQ ScreenBar Halo Wireless Controller LED', 2450000.00),
    ('WBD-006', 'Apple Magic Trackpad Multi-Touch Surface', 1890000.00),
    ('WBD-007', 'Sony WH-1000XM5 Noise Canceling Headphones', 4990000.00),
    ('WBD-008', 'Shure MV7X XLR Dynamic Podcast Microphone', 3200000.00),
    ('WBD-009', 'Elgato Stream Deck Plus Audio Mixer Studio', 3450000.00),
    ('WBD-010', 'Herman Miller Embody Ergonomic Office Chair', 24500000.00),
    ('WBD-011', 'Autonomous SmartDesk Pro Electric Standing Desk', 8900000.00),
    ('WBD-012', 'Samsung 990 PRO 2TB PCIe 4.0 NVMe SSD', 2750000.00),
    ('WBD-013', 'Anker Prime 27650mAh Power Bank 250W', 2100000.00),
    ('WBD-014', 'Audioengine A2+ Plus Wireless Desktop Speakers', 4150000.00),
    ('WBD-015', 'Orbitkey Desk Mat with Magnetic Cable Holder', 850000.00),
    ('WBD-016', 'LG DualUp 28-inch 16:18 SDQHD Ergo Monitor', 8400000.00),
    ('WBD-017', 'NuPhy Air75 V2 Ultra-Slim Wireless Keyboard', 1950000.00),
    ('WBD-018', 'Razer Viper V3 Pro Ultra-lightweight Mouse', 2390000.00),
    ('WBD-019', 'Rode PSA1+ Professional Studio Boom Arm', 1750000.00),
    ('WBD-020', 'Belkin MagSafe 3-in-1 Fast Wireless Charging', 1990000.00);

INSERT INTO inventory_movements (product_id, quantity_change, movement_type) VALUES
    (1, 50, 'initial_stock'),
    (2, 80, 'initial_stock'),
    (3, 20, 'initial_stock'),
    (4, 15, 'initial_stock'),
    (5, 40, 'initial_stock'),
    (6, 35, 'initial_stock'),
    (7, 25, 'initial_stock'),
    (8, 30, 'initial_stock'),
    (9, 20, 'initial_stock'),
    (10, 8, 'initial_stock'),
    (1, -5, 'customer_sale'),
    (2, -12, 'customer_sale'),
    (3, -2, 'customer_sale'),
    (4, -3, 'customer_sale'),
    (5, -6, 'customer_sale'),
    (6, -4, 'customer_sale'),
    (7, -8, 'customer_sale'),
    (8, -5, 'customer_sale'),
    (9, -3, 'customer_sale'),
    (10, -1, 'customer_sale'),
    (1, 20, 'restock_in'),
    (2, 30, 'restock_in'),
    (3, 10, 'restock_in'),
    (7, 15, 'restock_in'),
    (11, 10, 'initial_stock'),
    (12, 45, 'initial_stock');

INSERT INTO products (sku, name, price)
WITH RECURSIVE cte (n) AS (
    SELECT 21
    UNION ALL
    SELECT n + 1 FROM cte WHERE n < 1000
)
SELECT 
    CONCAT('WBD-', LPAD(n, 5, '0')),
    CONCAT('Demo Product Item ', n),
    ROUND(100000 + (RAND() * 5000000), 2)
FROM cte;

INSERT INTO inventory_movements (product_id, quantity_change, movement_type)
WITH RECURSIVE cte_inv (n) AS (
    SELECT 1
    UNION ALL
    SELECT n + 1 FROM cte_inv WHERE n < 2000
)
SELECT 
    ((n % 1000) + 1),
    CASE (n % 3)
        WHEN 0 THEN 50
        WHEN 1 THEN -10
        ELSE 25
    END,
    CASE (n % 3)
        WHEN 0 THEN 'restock_in'
        WHEN 1 THEN 'customer_sale'
        ELSE 'audit_adjustment'
    END
FROM cte_inv;

GRANT ALL PRIVILEGES ON *.* TO 'demo_user'@'%' WITH GRANT OPTION;
FLUSH PRIVILEGES;
