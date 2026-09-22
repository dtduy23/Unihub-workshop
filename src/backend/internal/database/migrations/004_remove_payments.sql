-- ==========================================
-- MIGRATION 004: Loại bỏ hoàn toàn bảng Payments
-- ==========================================

-- 1. Xóa ràng buộc và bảng payments
DROP TABLE IF EXISTS public.payments CASCADE;

-- 2. Xóa cột liên kết payment trên registrations
ALTER TABLE public.registrations DROP COLUMN IF EXISTS payment_transaction_id;

-- 3. Chuyển đổi các bản ghi cũ đang PENDING_PAYMENT về CANCELLED
UPDATE public.registrations 
SET status = 'CANCELLED' 
WHERE status = 'PENDING_PAYMENT';

