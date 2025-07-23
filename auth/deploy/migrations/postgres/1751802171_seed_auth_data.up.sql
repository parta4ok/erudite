BEGIN;

-- Создаем тестовых пользователей (пароль: password123)
-- Хеш для пароля "password123" с bcrypt cost=10
INSERT INTO auth.users (uid, name, password_hash, rights, contacts) VALUES
    ('1', 'admin', '$2a$10$ft9DCzVOqK1EzQ.tLgAAVOBG.89o0zjQqzWpqRrtKdcv1iEu/G84u',
     ARRAY['mentor', 'user:read', 'test:view_results'],
     '{"email": "ivan.petrov@example.com", "phone": "+7-900-123-45-67", "telegram": "@ivan_mentor"}'),
    ('2', 'Мария Сидорова', '$2a$10$ft9DCzVOqK1EzQ.tLgAAVOBG.89o0zjQqzWpqRrtKdcv1iEu/G84u',
     ARRAY['mentor', 'user:read', 'test:view_results'],
     '{"email": "maria.sidorova@example.com", "phone": "+7-900-234-56-78", "telegram": "@maria_mentor"}'),
    ('3', 'Алексей Иванов', '$2a$10$ft9DCzVOqK1EzQ.tLgAAVOBG.89o0zjQqzWpqRrtKdcv1iEu/G84u',
     ARRAY['student', 'test:take'],
     '{"email": "alexey.ivanov@example.com", "phone": "+7-900-345-67-89", "telegram": "@alexey_student"}'),
    ('4', 'Елена Козлова', '$2a$10$ft9DCzVOqK1EzQ.tLgAAVOBG.89o0zjQqzWpqRrtKdcv1iEu/G84u',
     ARRAY['student', 'test:take'],
     '{"email": "elena.kozlova@example.com", "phone": "+7-900-456-78-90", "telegram": "@elena_student"}'),
    ('5', 'Дмитрий Смирнов', '$2a$10$ft9DCzVOqK1EzQ.tLgAAVOBG.89o0zjQqzWpqRrtKdcv1iEu/G84u',
     ARRAY['student', 'test:take'],
     '{"email": "dmitry.smirnov@example.com", "phone": "+7-900-567-89-01", "telegram": "@dmitry_student"}'),
    ('6', 'Админ', '$2a$10$ft9DCzVOqK1EzQ.tLgAAVOBG.89o0zjQqzWpqRrtKdcv1iEu/G84u',
     ARRAY['admin', 'user:create', 'user:read', 'user:update', 'user:delete', 'system:admin'],
     '{"email": "admin@example.com", "phone": "+7-900-000-00-00"}');

END;
