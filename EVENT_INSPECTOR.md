# Event Inspector (chỉ đọc)

Event Inspector kết nối vào phiên Microsoft Edge đang chạy thông qua endpoint CDP cục bộ, bật chế độ quan sát `Network`/`Page`, ghi lại các WebSocket frame, che các thông tin xác thực có khả năng nhạy cảm, giải mã protobuf wire khi có thể và ghi timeline JSON. Công cụ không gọi `Runtime.evaluate`, không gửi lệnh game và không cung cấp API phát lệnh game.

## Chạy trên Windows

Khởi động Edge bằng profile riêng và cổng debug cục bộ. Nếu Edge đang dùng lại profile hiện tại, hãy đóng các cửa sổ Edge trước:

```powershell
msedge.exe --remote-debugging-port=9222 --user-data-dir="$env:TEMP\tytd-edge-inspector"
```

Build và chạy từ thư mục gốc của repository:

```powershell
go build -o .\build\event-inspector.exe .\cmd\event-inspector
.\build\event-inspector.exe -edge-cdp 127.0.0.1:9222 -duration 2m -out .\event-inspector.json
```

Mở trang game trong cửa sổ Edge đó, thực hiện lại event cần kiểm tra rồi dừng bằng `Ctrl+C`. Dù đã che dữ liệu, hãy vẫn xem file JSON là dữ liệu nhạy cảm. Khi hệ điều hành hỗ trợ, file output được tạo với quyền chỉ chủ sở hữu mới đọc được.

## Giới hạn

Bộ giải mã protobuf không phụ thuộc schema nên chỉ báo số field wire. Muốn có tên field, hãy thêm mapping descriptor cho message generated tương ứng trong `internal/proto`. Việc phân loại activity được giữ ở mức thận trọng; event type mới vẫn xuất hiện trong timeline và chỉ nên được đặt tên sau khi đã xem xét fixture.

## Bộ phân tích packet Go

`Player` và `EventPlayer` có trường tùy chọn `Inspector *eventinspector.Analyzer`. Khi được gắn bởi harness chẩn đoán/test offline, reader và writer WebSocket đều ghi metadata packet. Analyzer nhận diện 21001, 21036 và 21003, liên kết response 21003 với request theo `activityId` và ghi lại các đường dẫn JSON đã thay đổi. Báo cáo chỉ chứa độ dài và fingerprint SHA-256, không chứa byte packet gốc hoặc thông tin xác thực.

Chỉ tạo scaffold module sau khi đã xem xét fixture offline và xác nhận dữ liệu:

```powershell
go run .\cmd\event-module-skeleton -name new_event -verified -out .\player_event_new_event.go
```

Nếu không có `-verified`, lệnh sẽ thất bại. Scaffold chỉ là khung bất hoạt, không chứa sender hoặc packet encoder.
