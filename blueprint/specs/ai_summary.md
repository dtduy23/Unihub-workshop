# Đặc tả: AI Summary (Pipe-and-Filter)

## Mô tả
Tự động tóm tắt nội dung các file PDF giới thiệu sự kiện để sinh viên dễ đọc, sử dụng mô hình Google Gemini.

## Luồng chính (Pipe-and-Filter)
Dữ liệu văn bản như dòng chảy đi qua một đường ống chứa nhiều màng lọc:
1. **Filter 1 (PDF Extractor):** Đọc file PDF thô, trích xuất tất cả chữ viết thành một khối Text thuần.
2. **Filter 2 (Text Cleaner):** Loại bỏ các khoảng trắng thừa, xóa các ký tự điều khiển (Control characters), tối ưu độ dài.
3. **Filter 3 (AI Caller):** Xây dựng Prompt và gọi API Gemini 2.5 Flash thông qua **Circuit Breaker** để lấy bản tóm tắt.
4. Lưu bản tóm tắt thu được vào cột `ai_summary` của sự kiện trong Database.

## Kịch bản lỗi
- Mất kết nối API Gemini hoặc Hết Quota (Lỗi 429/500): Circuit Breaker mở mạch (OPEN), ngừng gọi Gemini. Các file upload lên trong thời gian này sẽ báo lỗi "AI hiện đang quá tải" và chỉ lưu nội dung thường, không làm treo server.
- File PDF quá dài: Filter 2 cắt bớt Text hoặc sử dụng tham số `MAX_TOKENS` ở Filter 3 để báo lỗi sớm, không nhồi nhét quá hạn mức của AI.

## Ràng buộc
- Quá trình gọi AI phải được cấu hình Timeout (VD: 10 giây). 

## Tiêu chí chấp nhận
- Có khả năng cấu hình thay thế dễ dàng Filter (Ví dụ: Từ gọi Gemini sang gọi OpenAI) mà không cần sửa logic các phần khác nhờ thiết kế Pipe-and-Filter. Đoạn văn bản đầu ra ngắn gọn, đúng ngữ pháp.
