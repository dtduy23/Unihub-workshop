# Đặc tả: Đăng ký Workshop & Xử lý tranh chấp ghế

## Mô tả
Cho phép sinh viên đăng ký giữ chỗ workshop. Yêu cầu xử lý mượt mà lượng truy cập 12,000 user trong vài phút đầu và tuyệt đối không bán lố quá 60 ghế/phòng.

## Luồng chính
1. Sinh viên bấm "Đăng ký". Client sinh ra một UUID (hoặc Backend tự tạo) làm Idempotency Key.
2. Request đi qua Redis Rate Limiter. Nếu Token Bucket báo cạn -> Trả về HTTP 429.
3. Request đi vào Redis SETNX để kiểm tra Idempotency Key. Nếu tồn tại -> Từ chối xử lý đúp.
4. Request đi qua Virtual Waiting Room (Redis ZSET). Nếu tải quá cao -> HTTP 202 Queued. Sinh viên ở màn hình Loading.
5. Khi hệ thống thoáng, Middleware đẩy Event Đăng Ký vào **RabbitMQ**, trả về HTTP 202 Accepted.
6. **Registration Worker** đọc message từ RabbitMQ. 
7. Mở Transaction PostgreSQL, chạy lệnh `SELECT available_seats FROM workshops WHERE id = $1 FOR UPDATE` (Pessimistic Lock).
8. Nếu `seats > 0` -> Ghi nhận bản ghi đăng ký, `UPDATE` giảm đi 1 ghế, `COMMIT` transaction.
9. Worker kích hoạt Notification Service để gửi Email cấp phát vé QR (Ký mã RSA).

## Kịch bản lỗi
- 2 Worker cùng xử lý 2 request mua chiếc vé cuối cùng: Pessimistic Lock sẽ ép 1 Worker phải đợi. Worker chạy trước mua thành công (seats = 0). Worker chạy sau thấy seats = 0 sẽ Rollback, trả về HTTP 400 "Đã hết chỗ". Đảm bảo tuyệt đối không âm vé.
- Đứt kết nối RabbitMQ: Cấu hình Persistent Queue. Message chưa xử lý xong sẽ nằm yên trên đĩa cứng, Worker khởi động lại sẽ xử lý tiếp, không mất đăng ký của sinh viên.

## Ràng buộc
- Đảm bảo tính ACID của CSDL trong khối transaction cập nhật vé.

## Tiêu chí chấp nhận
- Dùng công cụ Load Test bắn 12,000 request đăng ký cùng lúc vào 1 workshop có sức chứa 1,000 ghế. Kết thúc test, Database báo còn đúng 0 vé, số bản ghi đăng ký thành công đúng 1,000, 11,000 request bị từ chối hợp lệ.
