-- Synthetic local/test fixtures only. These values are not production data.
INSERT INTO users(id,email_canonical,email_original,nickname,display_name,password_hash,state,role,token_version,updated_at)
VALUES
('019a0000-0000-7000-8000-000000000001','admin@example.test','admin@example.test','admin','Local Admin','not-a-login-hash','active','admin',0,'2026-01-01T00:00:00Z'),
('019a0000-0000-7000-8000-000000000002','user@example.test','user@example.test','user','Local User','not-a-login-hash','active','user',0,'2026-01-01T00:00:00Z');

INSERT INTO place_kinds(id,key,label,sort_order,updated_at)
VALUES('019a0000-0000-7000-8000-000000000010','nature','Природа',1,'2026-01-01T00:00:00Z');

INSERT INTO place_collections(id,slug,title,subtitle,description,background_color,sort_order,status,published_at,updated_at)
VALUES('019a0000-0000-7000-8000-000000000020','mountains','Горы','Маршруты в горах','Синтетическая подборка','#223344',1,'published','2026-01-01T00:00:00Z','2026-01-01T00:00:00Z');
INSERT INTO place_collection_filters(id,collection_id,slug,title,sort_order,updated_at)
VALUES('019a0000-0000-7000-8000-000000000021','019a0000-0000-7000-8000-000000000020','easy','Легко',1,'2026-01-01T00:00:00Z');

INSERT INTO places(id,slug,title,kind_id,summary,description_markdown,status,sort_order,location_label,address_display_text,opening_hours_timezone,navigation_point,geometry,published_at,updated_at)
VALUES
('019a0000-0000-7000-8000-000000000100','sulakskiy-kanon','Сулакский каньон','019a0000-0000-7000-8000-000000000010','Синтетическое опубликованное место','## Описание','published',1,'Дубки','Республика Дагестан','Europe/Moscow',ST_SetSRID(ST_MakePoint(46.80,43.00),4326),ST_SetSRID(ST_MakePoint(46.80,43.00),4326),'2026-01-01T00:00:00Z','2026-01-01T00:00:00Z'),
('019a0000-0000-7000-8000-000000000101','draft-place','Черновик','019a0000-0000-7000-8000-000000000010','Готов к публикации',NULL,'draft',2,NULL,NULL,NULL,ST_SetSRID(ST_MakePoint(47.00,42.90),4326),NULL,NULL,'2026-01-01T00:00:00Z'),
('019a0000-0000-7000-8000-000000000102','draft-without-gallery','Без галереи','019a0000-0000-7000-8000-000000000010',NULL,NULL,'draft',3,NULL,NULL,NULL,ST_SetSRID(ST_MakePoint(47.10,42.80),4326),NULL,NULL,'2026-01-01T00:00:00Z'),
('019a0000-0000-7000-8000-000000000103','draft-without-navigation','Без координат','019a0000-0000-7000-8000-000000000010',NULL,NULL,'draft',4,NULL,NULL,NULL,NULL,NULL,NULL,'2026-01-01T00:00:00Z');

INSERT INTO place_media(id,place_id,kind,url,thumbnail_url,width,height,alt_text,sort_order,updated_at)
VALUES
('019a0000-0000-7000-8000-000000000110','019a0000-0000-7000-8000-000000000100','image','https://example.test/canyon.jpg',NULL,1600,900,'Каньон',1,'2026-01-01T00:00:00Z'),
('019a0000-0000-7000-8000-000000000111','019a0000-0000-7000-8000-000000000101','image','https://example.test/draft.jpg',NULL,1200,800,'Черновик',1,'2026-01-01T00:00:00Z'),
('019a0000-0000-7000-8000-000000000112','019a0000-0000-7000-8000-000000000103','image','https://example.test/no-nav.jpg',NULL,1200,800,'Без координат',1,'2026-01-01T00:00:00Z');
UPDATE places SET cover_media_id='019a0000-0000-7000-8000-000000000110' WHERE id='019a0000-0000-7000-8000-000000000100';
UPDATE places SET cover_media_id='019a0000-0000-7000-8000-000000000111' WHERE id='019a0000-0000-7000-8000-000000000101';
UPDATE places SET cover_media_id='019a0000-0000-7000-8000-000000000112' WHERE id='019a0000-0000-7000-8000-000000000103';

INSERT INTO place_collection_memberships(id,place_id,collection_id,sort_order)
VALUES('019a0000-0000-7000-8000-000000000120','019a0000-0000-7000-8000-000000000100','019a0000-0000-7000-8000-000000000020',1);
INSERT INTO place_collection_filter_memberships(id,membership_id,filter_id,sort_order)
VALUES('019a0000-0000-7000-8000-000000000121','019a0000-0000-7000-8000-000000000120','019a0000-0000-7000-8000-000000000021',1);

INSERT INTO place_badges(id,place_id,title,tone,sort_order,updated_at)
VALUES('019a0000-0000-7000-8000-000000000130','019a0000-0000-7000-8000-000000000100','Популярное','accent',1,'2026-01-01T00:00:00Z');
INSERT INTO place_tags(id,slug,title,updated_at)
VALUES('019a0000-0000-7000-8000-000000000131','canyon','Каньон','2026-01-01T00:00:00Z');
INSERT INTO place_tag_assignments(place_id,tag_id,sort_order)
VALUES('019a0000-0000-7000-8000-000000000100','019a0000-0000-7000-8000-000000000131',1);
INSERT INTO place_contacts(id,place_id,type,label,display_value,action_url,sort_order,updated_at)
VALUES('019a0000-0000-7000-8000-000000000132','019a0000-0000-7000-8000-000000000100','website','Сайт','example.test','https://example.test',1,'2026-01-01T00:00:00Z');
INSERT INTO place_opening_periods(id,place_id,open_day,open_minutes,close_day,close_minutes,sort_order,updated_at)
VALUES('019a0000-0000-7000-8000-000000000133','019a0000-0000-7000-8000-000000000100','monday',540,'monday',1080,1,'2026-01-01T00:00:00Z');
