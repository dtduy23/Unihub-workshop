# UniHub Workshop — Project Proposal

## Vấn đề
Hiện tại, Trường Đại học A đang sử dụng Google Form để quản lý đăng ký và email thủ công để thông báo cho "Tuần lễ kỹ năng và nghề nghiệp". Tuy nhiên, với 8-12 workshop song song mỗi ngày, Google Form bộc lộ nhiều điểm yếu chí mạng:
- Không kiểm soát được số lượng ghế thực tế khi có hàng ngàn sinh viên truy cập cùng lúc (oversell).
- Quy trình check-in tại cửa hội trường chậm chạp, thủ công, đặc biệt khó khăn tại các khu vực mất mạng.
- Thiếu hệ thống quản trị tập trung để tự động hóa việc tóm tắt thông tin sự kiện hay đồng bộ danh sách sinh viên.

## Mục tiêu
Xây dựng hệ thống UniHub Workshop Management số hóa toàn bộ quy trình:
- Chịu tải được **12,000 sinh viên** truy cập và đăng ký trong 10 phút đầu (60% dồn vào 3 phút đầu tiên) mà không sập hệ thống.
- Đảm bảo 100% không xảy ra tình trạng lố ghế (oversell) khi tranh chấp chỗ ngồi (60 ghế).
- Tự động hóa check-in bằng mã QR, hỗ trợ hoạt động hoàn toàn offline tại hầm mất mạng.
- Tự động tóm tắt nội dung PDF sự kiện bằng AI.

## Người dùng và nhu cầu
- **Sinh viên:** Xem lịch trình theo thời gian thực, đăng ký workshop, nhận mã QR check-in qua email/app. Quan trọng nhất: Hệ thống phản hồi nhanh, không lag khi tranh vé.
- **Ban tổ chức (Admin):** Tạo, cập nhật, hủy workshop. Có giao diện upload CSV đồng bộ sinh viên và upload PDF để AI tự động tóm tắt. Cần số liệu thống kê.
- **Nhân sự check-in (Staff):** Quét mã QR tại cửa. Cần một ứng dụng di động phản hồi ngay lập tức (dưới 1s), nhận diện được vé giả và hoạt động mượt mà ngay cả khi điện thoại không có Internet.

## Phạm vi
**Thuộc phạm vi:**
- Web App cho Sinh viên và Admin.
- Mobile App quét QR cho Staff.
- Hệ thống Backend xử lý tải cao (Message Queue, Rate Limiting, Waiting Room).
- Module tích hợp AI Summary (Google Gemini) và cơ chế Check-in Offline.

**Không thuộc phạm vi:**
- Hệ thống Payment Gateway thật (Sử dụng Mock Payment Endpoint thay thế để test tính ổn định).
- Hạ tầng Production thật (Sử dụng Docker Compose Local để mô phỏng).

## Rủi ro và ràng buộc
- **Tranh chấp chỗ ngồi:** Hàng trăm sinh viên tranh nhau 60 chỗ, dễ gây sai lệch dữ liệu nếu không khóa DB đúng cách.
- **Tải đột biến:** Lượng lớn kết nối đổ về có thể gây tràn bộ nhớ, cạn kiệt Connection Pool của Database.
- **Cổng thanh toán không ổn định:** Payment timeout có thể treo các thread hệ thống, sinh viên bị trừ tiền 2 lần khi retry.
- **Check-in offline:** Dữ liệu check-in lưu cục bộ có thể bị mất, đồng thời mã QR tự chế có thể qua mặt app offline.
- **Tích hợp một chiều CSV:** Hệ thống cũ export file 12,000 sinh viên, nhập thủ công sẽ tốn kém bộ nhớ và gây treo DB nếu thực hiện từng dòng (N+1 query).
