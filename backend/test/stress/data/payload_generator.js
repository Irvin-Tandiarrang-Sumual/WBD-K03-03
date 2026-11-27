export function randomInt(min, max) {
  return Math.floor(Math.random() * (max - min + 1)) + min;
}

export function randomString(length = 8) {
  const chars = 'abcdefghijklmnopqrstuvwxyz0123456789';
  let result = '';
  for (let i = 0; i < length; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length));
  }
  return result;
}

export function generateRandomReadQuery() {
  const queryGenerators = [
    () => `SELECT id, name, email FROM customers WHERE id = ${randomInt(1, 2000)};`,
    () => `SELECT customer_id, count(*), sum(total_amount) FROM orders WHERE customer_id BETWEEN ${randomInt(1, 500)} AND ${randomInt(501, 1500)} GROUP BY customer_id;`,
    () => `SELECT o.id, o.status, i.item_name, i.price FROM orders o JOIN order_items i ON o.id = i.order_id WHERE o.id = ${randomInt(1, 5000)};`,
    () => `SELECT count(*) as active_cnt FROM customers WHERE created_at <= NOW() - INTERVAL '${randomInt(1, 60)} minutes';`,
    () => `SELECT id, total_amount, status FROM orders WHERE status = '${['paid', 'pending', 'completed'][randomInt(0, 2)]}' ORDER BY id DESC LIMIT ${randomInt(5, 50)};`,
  ];
  return queryGenerators[randomInt(0, queryGenerators.length - 1)]();
}

export function generateExplainAnalyzeQuery() {
  const queryGenerators = [
    () => `EXPLAIN (ANALYZE, BUFFERS) SELECT c.id, c.name, count(o.id), avg(o.total_amount) FROM customers c LEFT JOIN orders o ON c.id = o.customer_id WHERE c.id > ${randomInt(1, 1000)} GROUP BY c.id, c.name;`,
    () => `EXPLAIN (ANALYZE, BUFFERS) SELECT * FROM orders WHERE total_amount > ${randomInt(100000, 2000000)} ORDER BY created_at DESC LIMIT 100;`,
  ];
  return queryGenerators[randomInt(0, queryGenerators.length - 1)]();
}

export function generateBatchUpdateSQL(tableName, columnName, valueExpression) {
  return {
    sqlUp: `UPDATE ${tableName} SET ${valueExpression} WHERE ${columnName} BETWEEN ? AND ?`,
    sqlDown: `UPDATE ${tableName} SET ${valueExpression} WHERE ${columnName} BETWEEN ? AND ?`,
    isBatch: true,
    batchColumn: columnName,
    batchSize: randomInt(50, 200),
  };
}
