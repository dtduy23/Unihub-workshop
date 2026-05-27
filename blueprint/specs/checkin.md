# Đặc tả: Check-in Offline (Mobile)

## Mô tả
Chức năng cho phép nhân sự (Staff) quét QR Code tại sự kiện để ghi nhận điểm danh. Đặc biệt, ứng dụng phải tiếp tục hoạt động và phát hiện vé giả mạo ngay cả khi khu vực sảnh mất mạng hoàn toàn.

## Luồng chính
1. **Chuẩn bị (Online):** App tải sẵn thông tin sự kiện cơ bản về lưu vào bộ nhớ đệm và sở hữu **RSA Public Key** nhúng sẵn trong code.
2. **Quét QR (Offline):** Staff đưa camera quét mã QR của sinh viên.
3. **Xác thực chữ ký số:** Mã QR là chuỗi JSON chứa `{ payload, signature }`. App trích xuất payload, tự băm mã SHA-256 và dùng Public Key để giải mã signature. Nếu trùng khớp -> Vé được cấp từ Backend. Nếu sai -> Vé giả mạo.
4. **Lưu trữ Offline:** App lưu ID sinh viên và ID sự kiện vào Database cục bộ (SQLite) với trạng thái `PENDING`. Hiển thị màu XANH báo hiệu thành công.
5. **Đồng bộ ngầm (Sync):** Background process (hook `useSync`) chạy mỗi 30s. Nếu máy chủ báo có kết nối Internet, nó sẽ bốc toàn bộ bản ghi `PENDING` thành 1 mảng JSON và đẩy Bulk lên API `/api/v1/checkin/sync`.
6. Backend trả về OK -> SQLite update trạng thái thành `SYNCED`.

## Kịch bản lỗi
- Quét vé giả do sinh viên tự in: Thuật toán RSA phát hiện không có chữ ký hợp lệ -> App báo màu ĐỎ (VÉ KHÔNG HỢP LỆ).
- Đồng bộ thất bại: Nếu server chết hoặc 3G rớt giữa chừng khi đang Sync, bản ghi SQLite vẫn giữ trạng thái `PENDING` và chờ vòng lặp 30s tiếp theo.

## Ràng buộc
- App tuyệt đối không được crash khi call API bị Network Error.

## Tiêu chí chấp nhận
- Rút cáp mạng / Tắt wifi máy Staff, vẫn có thể quét 100 vé thành công và báo hợp lệ. Khi bật mạng lên, 100 vé được đẩy lên trang Admin.
