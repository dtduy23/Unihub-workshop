-- Kích hoạt extension để tự động sinh UUID
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ==========================================
-- BẢNG 1: USERS (Người dùng hệ thống)
-- ==========================================
CREATE TABLE IF NOT EXISTS public.users (
  id uuid NOT NULL DEFAULT uuid_generate_v4(),
  user_id character varying NOT NULL UNIQUE, -- MSSV hoặc ID nhân sự
  password_hash character varying NOT NULL,
  full_name character varying NOT NULL,
  email character varying NOT NULL UNIQUE,
  phone character varying,
  role character varying NOT NULL DEFAULT 'STUDENT'::character varying,
  created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT users_pkey PRIMARY KEY (id)
);

-- Index tăng tốc truy vấn
CREATE INDEX IF NOT EXISTS idx_users_user_id ON users(user_id);

-- ==========================================
-- BẢNG 2: WORKSHOPS (Thông tin sự kiện)
-- ==========================================
CREATE TABLE IF NOT EXISTS public.workshops (
  id uuid NOT NULL DEFAULT uuid_generate_v4(),
  title character varying NOT NULL,
  description text,
  speaker text,
  room character varying NOT NULL,
  start_time timestamp with time zone NOT NULL,
  end_time timestamp with time zone NOT NULL,
  capacity integer NOT NULL CHECK (capacity > 0),
  available_seats integer NOT NULL,
  price numeric NOT NULL DEFAULT 0.00,
  summary text,
  room_layout_url character varying,
  status character varying NOT NULL DEFAULT 'PUBLISHED'::character varying,
  created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT workshops_pkey PRIMARY KEY (id)
);

-- ==========================================
-- BẢNG 3: REGISTRATIONS (Vé đăng ký)
-- ==========================================
CREATE TABLE IF NOT EXISTS public.registrations (
  id uuid NOT NULL DEFAULT uuid_generate_v4(),
  user_id uuid NOT NULL,
  workshop_id uuid NOT NULL,
  status character varying NOT NULL DEFAULT 'PENDING_PAYMENT'::character varying,
  ticket_signature text, -- Chữ ký số dùng để vẽ QR
  is_checked_in boolean NOT NULL DEFAULT false,
  created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
  payment_transaction_id character varying,
  CONSTRAINT registrations_pkey PRIMARY KEY (id),
  CONSTRAINT registrations_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE,
  CONSTRAINT registrations_workshop_id_fkey FOREIGN KEY (workshop_id) REFERENCES public.workshops(id) ON DELETE CASCADE,
  CONSTRAINT uq_user_workshop UNIQUE (user_id, workshop_id)
);

-- ==========================================
-- BẢNG 4: PAYMENTS (Giao dịch thanh toán)
-- ==========================================
CREATE TABLE IF NOT EXISTS public.payments (
  id uuid NOT NULL DEFAULT uuid_generate_v4(),
  registration_id uuid NOT NULL,
  transaction_id character varying NOT NULL UNIQUE,
  amount numeric NOT NULL,
  provider character varying NOT NULL,
  status character varying NOT NULL DEFAULT 'PENDING'::character varying,
  created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT payments_pkey PRIMARY KEY (id),
  CONSTRAINT payments_registration_id_fkey FOREIGN KEY (registration_id) REFERENCES public.registrations(id) ON DELETE CASCADE
);

-- ==========================================
-- BẢNG 5: NOTIFICATIONS (Thông báo)
-- ==========================================
CREATE TABLE IF NOT EXISTS public.notifications (
  id uuid NOT NULL DEFAULT uuid_generate_v4(),
  user_id uuid NOT NULL,
  registration_id uuid,
  channel character varying NOT NULL,
  title character varying NOT NULL,
  content text NOT NULL,
  status character varying NOT NULL DEFAULT 'PENDING'::character varying,
  event_id character varying NOT NULL,
  error_message text,
  created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
  sent_at timestamp with time zone,
  CONSTRAINT notifications_pkey PRIMARY KEY (id),
  CONSTRAINT notifications_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE,
  CONSTRAINT notifications_registration_id_fkey FOREIGN KEY (registration_id) REFERENCES public.registrations(id) ON DELETE SET NULL
);

-- ==========================================
-- BẢNG 6: IMPORT_JOBS (Lịch sử Import)
-- ==========================================
CREATE TABLE IF NOT EXISTS public.import_jobs (
  id uuid NOT NULL DEFAULT uuid_generate_v4(),
  file_name character varying NOT NULL,
  status character varying NOT NULL DEFAULT 'PROCESSING'::character varying,
  total_records integer DEFAULT 0,
  success_count integer DEFAULT 0,
  error_count integer DEFAULT 0,
  started_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
  completed_at timestamp with time zone,
  CONSTRAINT import_jobs_pkey PRIMARY KEY (id)
);

-- ==========================================
-- BẢNG 7: IMPORT_ERRORS (Lỗi Import)
-- ==========================================
CREATE TABLE IF NOT EXISTS public.import_errors (
  id uuid NOT NULL DEFAULT uuid_generate_v4(),
  job_id uuid NOT NULL,
  row_number integer NOT NULL,
  raw_data text,
  error_reason text NOT NULL,
  created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT import_errors_pkey PRIMARY KEY (id),
  CONSTRAINT import_errors_job_id_fkey FOREIGN KEY (job_id) REFERENCES public.import_jobs(id) ON DELETE CASCADE
);
