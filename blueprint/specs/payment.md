# Đặc tả: Luồng Thanh toán có phí (Fault Tolerance)

## Mô tả
Xử lý đăng ký các sự kiện có thu phí. Luồng này mô tả cách hệ thống chống đứt gãy dây chuyền (Cascading Failure) khi cổng thanh toán đối tác bị lỗi.

## Luồng chính
1. Sinh viên nhấn Thanh toán.
2. Hệ thống kiểm tra Idempotency Key trong Cache để chống tạo trùng đơn hàng.
3. Backend gọi API của Payment Gateway thông qua bộ bọc **Circuit Breaker**.
4. API phản hồi mã URL thanh toán. Sinh viên thanh toán thành công và vé được xuất.

## Kịch bản lỗi (Graceful Degradation)
- Cổng thanh toán chập chờn (Timeout liên tục): 
  - Thay vì để Backend chờ 30 giây rồi mới sập, làm kẹt cứng hàng ngàn Thread kết nối, **Circuit Breaker** theo dõi nếu lỗi > 50% trong 10 giây sẽ "Bật Cầu Dao" (OPEN State).
  - Lúc này, bất kỳ request đăng ký sự kiện *CÓ PHÍ* nào cũng lập tức nhận phản hồi HTTP 503 "Cổng thanh toán đang bảo trì" (Fail Fast) trong 1ms.
  - Các sinh viên đăng ký sự kiện *MIỄN PHÍ* vẫn đăng ký bình thường, vì không đi qua cầu dao. (Graceful Degradation).
  - Sau 30 giây làm nguội, Cầu Dao chuyển sang HALF-OPEN, cho lọt 1 vài request để dò đường. Nếu thành công, Cầu Dao đóng lại (CLOSED) và mọi thứ bình thường.

## Ràng buộc
- Thời gian Timeout cho API bên thứ ba không vượt quá 3s.

## Tiêu chí chấp nhận
- Giả lập tắt mạng đến server thanh toán. Circuit breaker kích hoạt, API đăng ký miễn phí vẫn đạt tốc độ < 50ms, hệ thống không treo.
