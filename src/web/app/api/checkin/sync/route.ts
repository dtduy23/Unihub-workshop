import { NextResponse } from 'next/server';

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export async function POST(request: Request) {
  try {
    const records = await request.json();

    console.log(`[Sync] Đang xử lý đồng bộ ${records.length} bản ghi...`);

    if (!records || records.length === 0) {
      return NextResponse.json({ status: 'success', synced_ids: [] });
    }

    // Lấy JWT token từ cookie để forward tới Go Backend
    const cookieHeader = request.headers.get('cookie') || '';
    const tokenMatch = cookieHeader.match(/unihub_token=([^;]*)/);
    const token = tokenMatch ? decodeURIComponent(tokenMatch[1]) : '';

    if (!token) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
    }

    // Forward toàn bộ request tới Go Backend API
    const res = await fetch(`${API_URL}/api/v1/checkin/sync`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`,
      },
      body: JSON.stringify(records),
    });

    const data = await res.json();
    return NextResponse.json(data, { status: res.status });
  } catch (error: any) {
    console.error('[Sync Fatal Error]:', error.message);
    return NextResponse.json({ error: 'Lỗi hệ thống' }, { status: 500 });
  }
}
