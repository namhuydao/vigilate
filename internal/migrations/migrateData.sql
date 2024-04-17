INSERT INTO public.users (id, first_name, last_name, user_active, access_level, email, password, created_at, updated_at, deleted_at)
       VALUES (1, 'Admin', 'User', 1, 3, 'admin@example.com', '$2a$12$F0IySlEPTRjXOY5l3mrl9.aEWJrpajLuVn3gKcZlXqLB9AqY0BB02', '2018-11-30 20:24:19.000000 +00:00', '2020-01-18 12:00:21.985541 +00:00', NULL);


INSERT INTO public.preferences (id, name, preference, created_at, updated_at) VALUES (1, 'monitoring_live', '0', '2020-06-26 07:49:33.648011 +00:00', '2020-06-26 07:49:33.648011 +00:00');
INSERT INTO public.preferences (id, name, preference, created_at, updated_at) VALUES (2, 'check_interval_amount', '3', '2020-06-26 07:49:33.648011 +00:00', '2020-06-26 07:49:33.648011 +00:00');
INSERT INTO public.preferences (id, name, preference, created_at, updated_at) VALUES (3, 'check_interval_unit', 'm', '2020-06-26 07:49:33.648011 +00:00', '2020-06-26 07:49:33.648011 +00:00');
INSERT INTO public.preferences (id, name, preference, created_at, updated_at) VALUES (4, 'notify_via_email', '0', '2020-06-26 07:49:33.648011 +00:00', '2020-06-26 07:49:33.648011 +00:00');

INSERT INTO public.services (id, service_name, active, icon, created_at, updated_at) VALUES (1, 'HTTP', 1, 'fas fa-server', '2024-04-11 02:20:08.000000', '2024-04-11 02:20:09.000000');

