#!/usr/bin/env python3
"""
Generate 12,000 realistic Vietnamese student records for UniHub Workshop.
Output: data/students_12k.csv (CSV format matching batch import pipeline)
Usage: python3 scripts/generate_12k_students.py
"""

import csv
import random
import os

# Vietnamese name components
HO = [
    "Nguyễn", "Trần", "Lê", "Phạm", "Hoàng", "Huỳnh", "Phan", "Vũ",
    "Võ", "Đặng", "Bùi", "Đỗ", "Hồ", "Ngô", "Dương", "Lý",
    "Mai", "Trịnh", "Đoàn", "Đinh", "Lâm", "Tạ", "Cao", "Hà",
]

TEN_DEM = [
    "Văn", "Thị", "Minh", "Hoàng", "Đức", "Quốc", "Thanh", "Bảo",
    "Hữu", "Ngọc", "Anh", "Phương", "Xuân", "Thu", "Hồng", "Kim",
    "Tuấn", "Thiện", "Gia", "Nhật", "Trung", "Quang", "Duy", "Hải",
]

TEN = [
    "An", "Bình", "Cường", "Duy", "Em", "Phong", "Giang", "Hùng",
    "Inh", "Khoa", "Khánh", "Linh", "Minh", "Nhật", "Oanh", "Phúc",
    "Quân", "Rạng", "Sơn", "Tùng", "Uyên", "Vinh", "Xuân", "Yến",
    "Hà", "Hương", "Lan", "Nga", "Thảo", "Trang", "Trinh", "Vân",
    "Đạt", "Huy", "Long", "Nam", "Tâm", "Thắng", "Trí", "Hiếu",
    "Bảo", "Kiệt", "Lộc", "Phát", "Tài", "Thành", "Toàn", "Trung",
]

# Email name mappings (ASCII versions for email)
TEN_ASCII = {
    "An": "an", "Bình": "binh", "Cường": "cuong", "Duy": "duy",
    "Em": "em", "Phong": "phong", "Giang": "giang", "Hùng": "hung",
    "Inh": "inh", "Khoa": "khoa", "Khánh": "khanh", "Linh": "linh",
    "Minh": "minh", "Nhật": "nhat", "Oanh": "oanh", "Phúc": "phuc",
    "Quân": "quan", "Rạng": "rang", "Sơn": "son", "Tùng": "tung",
    "Uyên": "uyen", "Vinh": "vinh", "Xuân": "xuan", "Yến": "yen",
    "Hà": "ha", "Hương": "huong", "Lan": "lan", "Nga": "nga",
    "Thảo": "thao", "Trang": "trang", "Trinh": "trinh", "Vân": "van",
    "Đạt": "dat", "Huy": "huy", "Long": "long", "Nam": "nam",
    "Tâm": "tam", "Thắng": "thang", "Trí": "tri", "Hiếu": "hieu",
    "Bảo": "bao", "Kiệt": "kiet", "Lộc": "loc", "Phát": "phat",
    "Tài": "tai", "Thành": "thanh", "Toàn": "toan", "Trung": "trung",
}

HO_ASCII = {
    "Nguyễn": "nguyen", "Trần": "tran", "Lê": "le", "Phạm": "pham",
    "Hoàng": "hoang", "Huỳnh": "huynh", "Phan": "phan", "Vũ": "vu",
    "Võ": "vo", "Đặng": "dang", "Bùi": "bui", "Đỗ": "do",
    "Hồ": "ho", "Ngô": "ngo", "Dương": "duong", "Lý": "ly",
    "Mai": "mai", "Trịnh": "trinh", "Đoàn": "doan", "Đinh": "dinh",
    "Lâm": "lam", "Tạ": "ta", "Cao": "cao", "Hà": "ha",
}

def generate_students(count=12000):
    """Generate student records."""
    students = []
    used_ids = set()
    
    # Start from student ID 31127100 to avoid collision with old test data
    base_id = 31127100
    
    for i in range(count):
        student_id = str(base_id + i)
        
        ho = random.choice(HO)
        ten_dem = random.choice(TEN_DEM)
        ten = random.choice(TEN)
        full_name = f"{ho} {ten_dem} {ten}"
        
        # Use student_id directly in email to guarantee uniqueness
        email = f"sv{student_id}@student.edu.vn"
        
        phone = f"09{random.randint(10000000, 99999999)}"
        password = "123456"
        role = "STUDENT"
        
        students.append([student_id, password, full_name, email, phone, role])
    
    return students


def main():
    script_dir = os.path.dirname(os.path.abspath(__file__))
    project_dir = os.path.dirname(script_dir)
    output_path = os.path.join(project_dir, "data", "students_12k.csv")
    
    print(f"🎓 Generating 12,000 student records...")
    students = generate_students(12000)
    
    with open(output_path, "w", newline="", encoding="utf-8") as f:
        writer = csv.writer(f)
        writer.writerow(["student_id", "password", "full_name", "email", "phone", "role"])
        writer.writerows(students)
    
    file_size = os.path.getsize(output_path) / (1024 * 1024)
    print(f"✅ Generated {len(students)} students → {output_path}")
    print(f"📦 File size: {file_size:.2f} MB")
    print(f"📋 Student ID range: {students[0][0]} → {students[-1][0]}")
    print(f"\n📌 To import via API:")
    print(f"   1. Login as admin:  curl -X POST http://localhost:8080/api/v1/auth/login -H 'Content-Type: application/json' -d '{{\"student_id\":\"ADMIN001\",\"password\":\"password123\"}}'")
    print(f"   2. Upload CSV:      curl -X POST http://localhost:8080/api/v1/admin/import/csv -H 'Authorization: Bearer <TOKEN>' -F 'file=@data/students_12k.csv'")
    print(f"\n📌 To import via batch scheduler:")
    print(f"   cp data/students_12k.csv data/imports/")


if __name__ == "__main__":
    main()
