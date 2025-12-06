ALTER SEQUENCE categories_id_seq RESTART WITH 1;

-- Категории
INSERT INTO categories (name, description) VALUES
('Протеин', 'Белковые добавки для роста мышц'),
('Батончики', 'Протеиновые батончики'),
('Креатин', 'Для силы и выносливости'),
('Печенье', 'Протеиновое печенье'),
('Чипсы', 'Натуральные протеиновые чипсы');

-- Протеин
INSERT INTO products (name, description, price, quantity, category_id, weight, flavor, servings) VALUES
('Whey Protein', 'Сывороточный протеин', 2500.00, 0, 1, '400', 'Шоколад', 20),
('Whey Protein', 'Сывороточный протеин', 5500.00, 0, 1, '1000', 'Шоколад', 50),
('Whey Protein', 'Сывороточный протеин', 2500.00, 0, 1, '400', 'Клубничный', 20),
('Whey Protein', 'Сывороточный протеин', 5500.00, 0, 1, '1000', 'Клубничный', 50),
('Whey Protein', 'Сывороточный протеин', 2500.00, 0, 1, '400', 'Карамельный', 20),
('Whey Protein', 'Сывороточный протеин', 5500.00, 0, 1, '1000', 'Карамельный', 50);

-- Батончики
INSERT INTO products (name, description, price, quantity, category_id, weight, flavor, servings) VALUES
('Protein Bar', 'Вкусный протеиновый батончик', 150.00, 0, 2, '40', 'Малина', 1),
('Protein Bar', 'Вкусный протеиновый батончик', 200.00, 0, 2, '60', 'Малина', 1),
('Protein Bar', 'Вкусный протеиновый батончик', 150.00, 0, 2, '40', 'Тирамису', 1),
('Protein Bar', 'Вкусный протеиновый батончик', 200.00, 0, 2, '60', 'Тирамису', 1),
('Protein Bar', 'Вкусный протеиновый батончик', 150.00, 0, 2, '40', 'Манго', 1),
('Protein Bar', 'Вкусный протеиновый батончик', 200.00, 0, 2, '60', 'Манго', 1);

-- Креатин
INSERT INTO products (name, description, price, quantity, category_id, weight, flavor, servings) VALUES
('Creatine Monohydrate', 'Чистый креатин моногидрат', 1200.00, 0, 3, '300', 'Классический', 60),
('Creatine Monohydrate', 'Чистый креатин моногидрат', 1800.00, 0, 3, '500', 'Классический', 100);

-- Печенье
INSERT INTO products (name, description, price, quantity, category_id, weight, flavor, servings) VALUES
('Protein Cookie', 'Полезное протеиновое печенье', 120.00, 0, 4, '100', 'Шоколад', 1),
('Protein Cookie', 'Полезное протеиновое печенье', 120.00, 0, 4, '100', 'Классическое', 1),
('Protein Cookie', 'Полезное протеиновое печенье', 120.00, 0, 4, '100', 'Корица', 1);

-- Чипсы
INSERT INTO products (name, description, price, quantity, category_id, weight, flavor, servings) VALUES
('Protein Chips', 'Натуральные протеиновые чипсы', 180.00, 0, 5, '80', 'Лук', 1),
('Protein Chips', 'Натуральные протеиновые чипсы', 250.00, 0, 5, '120', 'Лук', 1),
('Protein Chips', 'Натуральные протеиновые чипсы', 180.00, 0, 5, '80', 'Сыр', 1),
('Protein Chips', 'Натуральные протеиновые чипсы', 250.00, 0, 5, '120', 'Сыр', 1);

-- Пользователи
INSERT INTO users (telegram_id, username, first_name, phone, email, role) VALUES
(123456789, 'alex_admin', 'Алексей', '+79161234567', 'alex@example.com', 'admin'),
(987654321, 'maria_fit', 'Мария', '+79167654321', 'maria@example.com', 'user'),
(555666777, 'sport_ivan', 'Иван', '+79169998877', 'ivan@example.com', 'user'),
(111222333, 'anna_train', 'Анна', '+79165554433', 'anna@example.com', 'user'),
(444555666, 'max_power', 'Максим', '+79163332211', 'max@example.com', 'user'),
(444444444, 'alex_admin', 'Алексей', '+79161234567', 'alex@example.com', 'admin'),
(123123213, 'kolya', 'gsds', '+79167654321', 'maria@example.com', 'user'),
(435423445, 'durak', 'Ивgggан', '+79169998877', 'ivan@example.com', 'user'),
(776548902, 'trainings,ostea', 'ann', '+79165554433', 'anna@example.com', 'user'),
(450124042, 'gooder', 'ass', '+79163332211', 'max@example.com', 'admin');

-- Заказы
INSERT INTO orders (user_id, amount, status, created_at) VALUES
(2, 0, 'new', NOW()),
(3, 0, 'new', NOW()),
(4, 0, 'new', NOW() - INTERVAL '1 day'), 
(5, 0, 'new', NOW() - INTERVAL '2 days'), 
(1, 0, 'new', NOW() - INTERVAL '3 days'), 
(2, 0, 'new', NOW() - INTERVAL '4 days'),
(3, 0, 'new', NOW() - INTERVAL '5 days'),
(4, 0, 'new', NOW() - INTERVAL '6 days'),
(5, 0, 'new', NOW() - INTERVAL '7 days'),
(1, 0, 'new', NOW() - INTERVAL '8 days');

INSERT INTO order_items (order_id, product_id, quantity, price) VALUES
(5, 1, 2, 2500.00),
(5, 7, 3, 150.00),
(5, 13, 1, 1200.00);

INSERT INTO order_items (order_id, product_id, quantity, price) VALUES
(10, 3, 1, 2500.00),
(10, 9, 2, 150.00),
(10, 17, 4, 180.00);

UPDATE orders SET amount = (
    SELECT SUM(quantity * price) 
    FROM order_items 
    WHERE order_id = 5
) WHERE id = 5;

UPDATE orders SET amount = (
    SELECT SUM(quantity * price) 
    FROM order_items 
    WHERE order_id = 10
) WHERE id = 10;


