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
    ('WBD-001', 'Mechanical Keyboard', 875000.00),
    ('WBD-002', 'Wireless Mouse', 325000.00);

INSERT INTO inventory_movements (product_id, quantity_change, movement_type) VALUES
    (1, 20, 'stock_in'),
    (2, 35, 'stock_in');

-- Grant all privileges to demo_user so dynamic database creation is permitted
GRANT ALL PRIVILEGES ON *.* TO 'demo_user'@'%' WITH GRANT OPTION;
FLUSH PRIVILEGES;
