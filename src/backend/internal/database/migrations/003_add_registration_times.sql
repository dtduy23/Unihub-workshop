-- Thêm cột registration_start_time và registration_end_time
ALTER TABLE workshops ADD COLUMN registration_start_time timestamp with time zone;
ALTER TABLE workshops ADD COLUMN registration_end_time timestamp with time zone;

-- Cập nhật dữ liệu cũ (Fallback value)
UPDATE workshops SET 
  registration_start_time = created_at,
  registration_end_time = start_time
WHERE registration_start_time IS NULL;

-- Bắt buộc NOT NULL sau khi đã cập nhật dữ liệu cũ
ALTER TABLE workshops ALTER COLUMN registration_start_time SET NOT NULL;
ALTER TABLE workshops ALTER COLUMN registration_end_time SET NOT NULL;
