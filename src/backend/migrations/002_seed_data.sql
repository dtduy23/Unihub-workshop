-- Seed data for UniHub Workshop
-- Password: 123456 (bcrypt hashed with cost 10)

-- Users
INSERT INTO users (user_id, password_hash, full_name, email, phone, role) VALUES
('21127001', '$2a$10$x6.ouOf.LBM3OogQNHcKZe/De3W9uivpBZ2Ty1mt3SQxBXq9Jb2JG', 'Nguyễn Văn An', 'an.nguyen@student.edu.vn', '0901234001', 'STUDENT'),
('21127002', '$2a$10$x6.ouOf.LBM3OogQNHcKZe/De3W9uivpBZ2Ty1mt3SQxBXq9Jb2JG', 'Trần Thị Bình', 'binh.tran@student.edu.vn', '0901234002', 'STUDENT'),
('21127003', '$2a$10$x6.ouOf.LBM3OogQNHcKZe/De3W9uivpBZ2Ty1mt3SQxBXq9Jb2JG', 'Lê Hoàng Cường', 'cuong.le@student.edu.vn', '0901234003', 'STUDENT'),
('21127004', '$2a$10$x6.ouOf.LBM3OogQNHcKZe/De3W9uivpBZ2Ty1mt3SQxBXq9Jb2JG', 'Phạm Minh Duy', 'duy.pham@student.edu.vn', '0901234004', 'STUDENT'),
('21127005', '$2a$10$x6.ouOf.LBM3OogQNHcKZe/De3W9uivpBZ2Ty1mt3SQxBXq9Jb2JG', 'Hoàng Thị Em', 'em.hoang@student.edu.vn', '0901234005', 'STUDENT'),
('admin', '$2a$10$x6.ouOf.LBM3OogQNHcKZe/De3W9uivpBZ2Ty1mt3SQxBXq9Jb2JG', 'System Administrator', 'admin@unihub.vn', '0909999001', 'ADMIN'),
('staff01', '$2a$10$x6.ouOf.LBM3OogQNHcKZe/De3W9uivpBZ2Ty1mt3SQxBXq9Jb2JG', 'Trần Staff', 'staff@unihub.vn', '0909999002', 'STAFF')
ON CONFLICT (user_id) DO NOTHING;

-- Workshops (future dates, various capacities for load testing)
INSERT INTO workshops (title, description, speaker, room, start_time, end_time, capacity, available_seats, price, status) VALUES
('Kỹ năng CV và Phỏng vấn', 'Workshop hướng dẫn viết CV chuyên nghiệp', 'ThS. Nguyễn Minh Tuấn', 'A101', NOW() + interval '7 day', NOW() + interval '7 day 2 hour', 200, 200, 0.00, 'PUBLISHED'),
('Lập trình Python cho người mới', 'Khóa học cơ bản về Python', 'TS. Trần Văn Hùng', 'B201', NOW() + interval '8 day', NOW() + interval '8 day 3 hour', 5000, 5000, 50000.00, 'PUBLISHED'),
('Design Thinking Workshop', 'Phương pháp tư duy thiết kế', 'MBA. Lê Thu Hà', 'C301', NOW() + interval '9 day', NOW() + interval '9 day 3 hour', 50, 50, 0.00, 'PUBLISHED'),
('Cloud Computing with AWS', 'Thực hành triển khai trên AWS', 'AWS SA Trần Minh', 'B201', NOW() + interval '10 day', NOW() + interval '10 day 4 hour', 10000, 10000, 0.00, 'PUBLISHED');
