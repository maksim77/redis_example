CREATE TABLE IF NOT EXISTS users (
    id INT PRIMARY KEY,
    name VARCHAR(100),
    age INT
);

INSERT INTO users (id, name, age) 
VALUES (1, 'Test User', 30)
ON CONFLICT DO NOTHING;