# Đặc tả: Phân quyền hệ thống (Auth & RBAC)

## Mô tả
Tính năng cấp phát JSON Web Token (JWT) và bảo vệ các luồng nghiệp vụ dựa trên Role-Based Access Control (RBAC).

## Luồng chính
1. Người dùng gửi Request Login (gồm MSSV/Username và Password) đến Backend.
2. Backend băm Password bằng thuật toán `bcrypt` và đối chiếu với hash lưu trong Database PostgreSQL.
3. Nếu khớp, Backend sinh một JWT có thời hạn 24 giờ. Payload JWT chứa `user_id` và `role` (`STUDENT`, `STAFF`, hoặc `ADMIN`).
4. Client nhận JWT và lưu trữ. Từ lần gọi API sau, Client đính kèm JWT vào Header `Authorization: Bearer <token>`.
5. Middleware trên Backend sẽ decode JWT. Dựa trên route được gọi, Middleware kiểm tra trường `role` có khớp với quyền truy cập hay không. Nếu không -> HTTP 403 Forbidden.

## Kịch bản lỗi
- User nhập sai password: HTTP 401 Unauthorized.
- JWT hết hạn hoặc bị giả mạo: Middleware từ chối, HTTP 401. Đẩy người dùng về trang Login.

## Ràng buộc
- JWT phải được ký bằng một `JWT_SECRET` mạnh. Mật khẩu trong DB bắt buộc phải băm bằng `bcrypt`.

## Tiêu chí chấp nhận
- Không ai ngoài ADMIN có thể truy cập `/api/v1/admin/*`.
- STAFF không thể đăng ký hoặc tạo workshop.
