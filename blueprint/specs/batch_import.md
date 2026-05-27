# Đặc tả: Tích hợp một chiều CSV (Batch Import)

## Mô tả
Hệ thống lấy thông tin sinh viên từ tập tin CSV xuất ban đêm của nhà trường để xác thực. Phải xử lý lượng lớn dữ liệu mà không làm kẹt DB.

## Luồng chính
1. Admin chọn file CSV (ví dụ: `students_12k.csv`) tải lên giao diện Web Admin.
2. File được parse thành mảng các đối tượng Sinh Viên trên RAM của server.
3. Password của từng sinh viên được băm bằng thuật toán `bcrypt` (để sinh viên có thể tự đăng nhập sau này).
4. Do có tới 12,000 bản ghi, hệ thống không dùng vòng lặp `INSERT` từng dòng. API cắt mảng thành các **Chunks (Lô)** nhỏ (ví dụ 1,000 dòng/lô).
5. Sử dụng hàm `UNNEST` của PostgreSQL để đẩy toàn bộ 1,000 dòng này vào DB chỉ bằng 1 câu truy vấn SQL: `INSERT INTO users ... ON CONFLICT (user_id) DO UPDATE SET ...` (Bulk Upsert).

## Kịch bản lỗi
- Dữ liệu rác/File hỏng: Quá trình Validate báo lỗi trước khi chạm vào DB.
- Trùng lặp dữ liệu: Do dùng `ON CONFLICT DO UPDATE`, nếu Admin lỡ import 1 file 2 lần, hệ thống sẽ thực thi luồng UPDATE một cách an toàn mà không sinh ra lỗi duplicate key, duy trì tính Luỹ Đẳng (Idempotent).

## Ràng buộc
- Quá trình xử lý file 12,000 dòng không được làm gián đoạn (block) các API đăng ký đang chạy.

## Tiêu chí chấp nhận
- Import thành công 12,000 dòng trong dưới 5 giây. Không sinh lỗi tràn bộ nhớ.
