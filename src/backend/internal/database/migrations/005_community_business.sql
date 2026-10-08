-- Existing accounts retain their role; unknown legacy roles must be resolved explicitly.
ALTER TABLE users ADD COLUMN auth_version integer NOT NULL DEFAULT 1;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('STUDENT','STAFF','BUSINESS','ADMIN'));

CREATE TABLE password_resets (
  token_hash text PRIMARY KEY, user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  expires_at timestamptz NOT NULL, created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE companies (
  id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
  owner_user_id uuid NOT NULL UNIQUE REFERENCES users(id),
  name text NOT NULL, slug text NOT NULL UNIQUE,
  description text NOT NULL DEFAULT '', industry text NOT NULL DEFAULT '',
  website text NOT NULL DEFAULT '', address text NOT NULL DEFAULT '', contact text NOT NULL DEFAULT '',
  logo_url text NOT NULL DEFAULT '', cover_url text NOT NULL DEFAULT '',
  status text NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING','APPROVED','REJECTED','SUSPENDED')),
  review_reason text NOT NULL DEFAULT '', created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE workshops ADD COLUMN company_id uuid REFERENCES companies(id);
ALTER TABLE workshops ADD COLUMN created_by uuid REFERENCES users(id);
ALTER TABLE workshops ADD COLUMN cover_url text NOT NULL DEFAULT '';
ALTER TABLE workshops ADD COLUMN audience text NOT NULL DEFAULT '';
ALTER TABLE workshops ADD COLUMN benefits text NOT NULL DEFAULT '';
ALTER TABLE workshops ADD COLUMN preparation text NOT NULL DEFAULT '';
ALTER TABLE workshops ADD COLUMN agenda text NOT NULL DEFAULT '';
ALTER TABLE workshops ADD COLUMN format text NOT NULL DEFAULT 'OFFLINE' CHECK (format IN ('OFFLINE','ONLINE','HYBRID'));
ALTER TABLE workshops ADD COLUMN review_reason text NOT NULL DEFAULT '';
ALTER TABLE workshops ADD COLUMN updated_at timestamptz NOT NULL DEFAULT now();
ALTER TABLE workshops ADD CONSTRAINT workshops_status_check CHECK (status IN ('DRAFT','PENDING_REVIEW','REJECTED','PUBLISHED','CLOSED','CANCELLED','DELETED'));
CREATE INDEX workshops_company_schedule ON workshops(company_id,start_time,id);

CREATE TABLE workshop_revisions (
  id uuid PRIMARY KEY DEFAULT uuid_generate_v4(), workshop_id uuid NOT NULL REFERENCES workshops(id),
  requested_by uuid NOT NULL REFERENCES users(id), payload jsonb NOT NULL,
  status text NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING','APPROVED','REJECTED')),
  reason text NOT NULL DEFAULT '', cancellation boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX workshop_revision_pending ON workshop_revisions(workshop_id) WHERE status='PENDING';
CREATE TABLE workshop_staff (
  workshop_id uuid REFERENCES workshops(id) ON DELETE CASCADE, user_id uuid REFERENCES users(id) ON DELETE CASCADE,
  PRIMARY KEY(workshop_id,user_id)
);

ALTER TABLE registrations ADD COLUMN checked_in_at timestamptz;
UPDATE registrations SET checked_in_at=updated_at WHERE is_checked_in;
ALTER TABLE registrations ALTER COLUMN status SET DEFAULT 'PROCESSING';
CREATE TABLE registration_requests (
  id uuid PRIMARY KEY, user_id uuid NOT NULL REFERENCES users(id), workshop_id uuid NOT NULL REFERENCES workshops(id),
  published boolean NOT NULL DEFAULT false, status text NOT NULL DEFAULT 'PROCESSING', message text NOT NULL DEFAULT '',
  registration_id uuid REFERENCES registrations(id), attempts integer NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX registration_request_active ON registration_requests(user_id,workshop_id) WHERE status='PROCESSING';

CREATE TABLE media_assets (
  id uuid PRIMARY KEY DEFAULT uuid_generate_v4(), owner_user_id uuid NOT NULL REFERENCES users(id),
  mime text NOT NULL, size_bytes bigint NOT NULL, created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE posts (
  id uuid PRIMARY KEY DEFAULT uuid_generate_v4(), author_user_id uuid NOT NULL REFERENCES users(id),
  company_id uuid REFERENCES companies(id), workshop_id uuid REFERENCES workshops(id),
  kind text NOT NULL DEFAULT 'COMMUNITY' CHECK (kind IN ('COMMUNITY','WORKSHOP_ANNOUNCEMENT')),
  title text NOT NULL, content text NOT NULL, topic text NOT NULL DEFAULT '',
  media_ids uuid[] NOT NULL DEFAULT '{}',
  status text NOT NULL DEFAULT 'DRAFT' CHECK(status IN ('DRAFT','PUBLISHED','HIDDEN','DELETED')),
  created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), published_at timestamptz,
  CHECK(kind<>'WORKSHOP_ANNOUNCEMENT' OR workshop_id IS NOT NULL)
);
CREATE INDEX posts_feed ON posts(published_at DESC,id DESC) WHERE status='PUBLISHED';
CREATE INDEX posts_company ON posts(company_id,published_at DESC);
CREATE TABLE post_likes (post_id uuid REFERENCES posts(id) ON DELETE CASCADE,user_id uuid REFERENCES users(id) ON DELETE CASCADE,PRIMARY KEY(post_id,user_id));
CREATE TABLE post_bookmarks (post_id uuid REFERENCES posts(id) ON DELETE CASCADE,user_id uuid REFERENCES users(id) ON DELETE CASCADE,PRIMARY KEY(post_id,user_id));
CREATE TABLE company_follows (company_id uuid REFERENCES companies(id) ON DELETE CASCADE,user_id uuid REFERENCES users(id) ON DELETE CASCADE,PRIMARY KEY(company_id,user_id));
CREATE TABLE comments (
  id uuid PRIMARY KEY DEFAULT uuid_generate_v4(), post_id uuid NOT NULL REFERENCES posts(id),
  author_user_id uuid NOT NULL REFERENCES users(id), parent_id uuid REFERENCES comments(id), content text NOT NULL,
  status text NOT NULL DEFAULT 'PUBLISHED' CHECK(status IN ('PUBLISHED','HIDDEN','DELETED')),
  created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX comments_post ON comments(post_id,created_at,id);
CREATE TABLE content_reports (
  id uuid PRIMARY KEY DEFAULT uuid_generate_v4(), reporter_id uuid NOT NULL REFERENCES users(id),
  post_id uuid REFERENCES posts(id), comment_id uuid REFERENCES comments(id), reason text NOT NULL,
  status text NOT NULL DEFAULT 'PENDING' CHECK(status IN ('PENDING','RESOLVED')),
  resolution text NOT NULL DEFAULT '', created_at timestamptz NOT NULL DEFAULT now(),
  CHECK((post_id IS NULL) <> (comment_id IS NULL))
);
CREATE UNIQUE INDEX reports_unique_post ON content_reports(reporter_id,post_id) WHERE post_id IS NOT NULL AND status='PENDING';
CREATE UNIQUE INDEX reports_unique_comment ON content_reports(reporter_id,comment_id) WHERE comment_id IS NOT NULL AND status='PENDING';
CREATE TABLE moderation_actions (
  id uuid PRIMARY KEY DEFAULT uuid_generate_v4(), actor_id uuid NOT NULL REFERENCES users(id),
  target_id uuid NOT NULL, action text NOT NULL, reason text NOT NULL, created_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE notifications ADD COLUMN link text NOT NULL DEFAULT '';
-- The existing repository already relies on this constraint for idempotent notifications.
DELETE FROM notifications a USING notifications b WHERE a.event_id=b.event_id AND a.channel=b.channel AND a.id>b.id;
CREATE UNIQUE INDEX IF NOT EXISTS notifications_event_channel ON notifications(event_id,channel);

-- Preserve staff access to existing workshops; new workshops require explicit assignment.
INSERT INTO workshop_staff(workshop_id,user_id) SELECT w.id,u.id FROM workshops w CROSS JOIN users u WHERE u.role='STAFF';

-- Durable email intent, committed with the registration. Reset secrets are excluded.
CREATE TABLE notification_outbox (
 event_id text PRIMARY KEY, payload jsonb NOT NULL,
 published boolean NOT NULL DEFAULT false, created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX registration_request_outbox ON registration_requests(created_at) WHERE NOT published AND status='PROCESSING';
CREATE INDEX notification_outbox_pending ON notification_outbox(created_at) WHERE NOT published;
