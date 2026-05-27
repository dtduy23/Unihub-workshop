import { NextResponse } from 'next/server';
import { supabase } from '@/lib/supabase';

export async function POST(request: Request) {
  try {
    const records = await request.json(); 
    
    console.log(`[Sync] Đang xử lý đồng bộ ${records.length} bản ghi...`);

    if (!records || records.length === 0) {
      return NextResponse.json({ status: 'success', synced_ids: [] });
    }

    const synced_ids: string[] = [];

    for (const rec of records) {
      let targetUUID = rec.user_uuid;

      // TRƯỜNG HỢP MÃ CŨ: Nếu UUID bị undefined hoặc rỗng, thực hiện tra cứu MSSV (Fallback)
      if (!targetUUID || targetUUID === 'undefined' || targetUUID === '') {
        console.log(`[Sync Fallback] Đang tra cứu UUID cho mã cũ của SV: ${rec.student_id}`);
        const { data: userData } = await supabase
          .from('users')
          .select('id')
          .eq('user_id', rec.student_id.trim())
          .single();
        
        if (userData) {
          targetUUID = userData.id;
        } else {
          console.error(`[Sync Error] Không tìm thấy SV ${rec.student_id} trong hệ thống.`);
          continue;
        }
      }

      // THỰC HIỆN UPDATE
      const { data, error } = await supabase
        .from('registrations')
        .update({ 
          is_checked_in: true,
          updated_at: new Date(rec.scanned_at).toISOString()
        })
        .eq('user_id', targetUUID)
        .eq('workshop_id', rec.workshop_id.trim())
        .select();

      if (error) {
        console.error(`[Sync Error] Lỗi DB cho SV ${rec.student_id}:`, error.message);
      } else if (!data || data.length === 0) {
        console.warn(`[Sync Warn] SV ${rec.student_id} chưa đăng ký workshop này.`);
      } else {
        console.log(`[Sync Success] Đã điểm danh SV ${rec.student_id}`);
        synced_ids.push(rec.id);
      }
    }
    
    return NextResponse.json({
      status: 'success',
      synced_ids: synced_ids,
      message: `Đồng bộ thành công ${synced_ids.length}/${records.length} bản ghi.`
    });
  } catch (error: any) {
    console.error('[Sync Fatal Error]:', error.message);
    return NextResponse.json({ error: 'Lỗi hệ thống' }, { status: 500 });
  }
}
