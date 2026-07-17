INSERT INTO customer_profiles (id, business_id, unified_name, customer_tier, semantic_summary) VALUES
(gen_random_uuid(), '3b825f22-73a8-487f-90d1-794fdf3e81df', 'Alice Johnson', 'gold', 'Frequent buyer'),
(gen_random_uuid(), '3b825f22-73a8-487f-90d1-794fdf3e81df', 'Marcus Wright', 'silver', 'New customer'),
(gen_random_uuid(), '3b825f22-73a8-487f-90d1-794fdf3e81df', 'Sarah Connor', 'standard', 'Prefers WhatsApp'),
(gen_random_uuid(), '3b825f22-73a8-487f-90d1-794fdf3e81df', 'Elena Rostova', 'standard', 'Recently cancelled an order'),
(gen_random_uuid(), '3b825f22-73a8-487f-90d1-794fdf3e81df', 'James Holden', 'gold', 'Prefers Messenger');

INSERT INTO orders (id, business_id, customer_id, total_price, status, source, notes, created_at, updated_at) VALUES
(gen_random_uuid(), '3b825f22-73a8-487f-90d1-794fdf3e81df', (SELECT id FROM customer_profiles WHERE unified_name = 'Alice Johnson' LIMIT 1), 124.50, 'completed', 'portal', '', '2026-07-17 09:24:00', '2026-07-17 09:24:00'),
(gen_random_uuid(), '3b825f22-73a8-487f-90d1-794fdf3e81df', (SELECT id FROM customer_profiles WHERE unified_name = 'Marcus Wright' LIMIT 1), 45.00, 'pending', 'portal', '', '2026-07-17 08:12:00', '2026-07-17 08:12:00'),
(gen_random_uuid(), '3b825f22-73a8-487f-90d1-794fdf3e81df', (SELECT id FROM customer_profiles WHERE unified_name = 'Sarah Connor' LIMIT 1), 399.99, 'pending', 'agent', '', '2026-07-16 18:45:00', '2026-07-16 18:45:00'),
(gen_random_uuid(), '3b825f22-73a8-487f-90d1-794fdf3e81df', (SELECT id FROM customer_profiles WHERE unified_name = 'Alice Johnson' LIMIT 1), 89.95, 'completed', 'agent', '', '2026-07-16 14:20:00', '2026-07-16 14:20:00'),
(gen_random_uuid(), '3b825f22-73a8-487f-90d1-794fdf3e81df', (SELECT id FROM customer_profiles WHERE unified_name = 'Elena Rostova' LIMIT 1), 210.00, 'cancelled', 'portal', '', '2026-07-15 11:05:00', '2026-07-15 11:05:00'),
(gen_random_uuid(), '3b825f22-73a8-487f-90d1-794fdf3e81df', (SELECT id FROM customer_profiles WHERE unified_name = 'James Holden' LIMIT 1), 15.50, 'completed', 'agent', '', '2026-07-15 09:30:00', '2026-07-15 09:30:00'),
(gen_random_uuid(), '3b825f22-73a8-487f-90d1-794fdf3e81df', (SELECT id FROM customer_profiles WHERE unified_name = 'James Holden' LIMIT 1), 550.00, 'pending', 'agent', '', '2026-07-14 16:15:00', '2026-07-14 16:15:00');
