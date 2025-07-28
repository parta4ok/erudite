BEGIN;

-- Создаем тестовых пользователей (пароль: password123)
-- Хеш для пароля "password123" с bcrypt cost=10
INSERT INTO auth.users (uid, name, password_hash, rights, contacts) VALUES
    ('1', 'admin@kvs.ru', '$2a$10$ft9DCzVOqK1EzQ.tLgAAVOBG.89o0zjQqzWpqRrtKdcv1iEu/G84u',
     ARRAY['admin', 'add_user', 'delete_user', 'view_topic_list', 'start_session', 'complete_session', 'view_completed_sessions'],
     '{"email": "admin@kvs.ru", "phone": "+7-900-123-45-67", "telegram": "@admin_kvs"}'),
    ('2', 'mentor1@kvs.ru', '$2a$10$ft9DCzVOqK1EzQ.tLgAAVOBG.89o0zjQqzWpqRrtKdcv1iEu/G84u',
     ARRAY['mentor', 'view_topic_list', 'start_session', 'complete_session', 'view_completed_sessions'],
     '{"email": "mentor1@kvs.ru", "phone": "+7-900-234-56-78", "telegram": "@maria_mentor"}'),
    ('3', 'john-doe@kvs.ru', '$2a$10$ft9DCzVOqK1EzQ.tLgAAVOBG.89o0zjQqzWpqRrtKdcv1iEu/G84u',
     ARRAY['student', 'view_topic_list', 'start_session', 'complete_session'],
     '{"email": "john-doe@kvs.ru", "phone": "+7-900-000-00-00", "telegram": "@JD_super_pupper"}');
END;
