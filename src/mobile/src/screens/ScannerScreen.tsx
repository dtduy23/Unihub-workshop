import React, { useState, useEffect, useRef } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, Dimensions, Vibration, ActivityIndicator } from 'react-native';
import { CameraView, useCameraPermissions, BarcodeScanningResult } from 'expo-camera';
import { X } from 'lucide-react-native';
import { saveCheckinLocal, isStudentCheckedIn } from '../services/storage';
import { verifyTicket } from '../services/crypto';
import * as Haptics from 'expo-haptics';

const { width, height } = Dimensions.get('window');

interface ScannerScreenProps {
  currentWorkshop: { id: string, title: string, room: string };
  onClose: () => void;
  onScanSuccess: (data: any) => void;
}

export default function ScannerScreen({ currentWorkshop, onClose, onScanSuccess }: ScannerScreenProps) {
  const [permission, requestPermission] = useCameraPermissions();
  const [scanStatus, setScanStatus] = useState<'idle' | 'processing' | 'success' | 'error' | 'warning'>('idle');
  const [message, setMessage] = useState('');
  const [count, setCount] = useState(45);

  const isLocked = useRef(false);

  if (!permission) return <View />;
  if (!permission.granted) {
    return (
      <View style={styles.center}>
        <Text style={styles.text}>Cần quyền truy cập Camera để quét mã</Text>
        <TouchableOpacity onPress={requestPermission} style={styles.btn}>
          <Text style={styles.btnText}>Cấp quyền</Text>
        </TouchableOpacity>
      </View>
    );
  }

  const handleBarCodeScanned = async (result: BarcodeScanningResult) => {
    const { data, bounds } = result;

    // 1. CHẶN QUÉT LIÊN TỤC
    if (isLocked.current || scanStatus !== 'idle') return;

    // 2. KIỂM TRA VÙNG QUÉT (GIỮA MÀN HÌNH)
    if (bounds) {
      const { origin, size } = bounds;
      const qrCenterY = origin.y + size.height / 2;
      // Chỉ chấp nhận nếu nằm trong khoảng 20% - 80% chiều cao màn hình
      if (qrCenterY < height * 0.2 || qrCenterY > height * 0.8) return;
    }

    // KHÓA VÀ RUNG PHẢN HỒI
    isLocked.current = true;
    setScanStatus('processing');
    try {
      await Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Heavy);
    } catch (e) {
      Vibration.vibrate(100);
    }

    try {
      // 3. GIẢI MÃ JSON
      console.log(`📷 [Scanner] Raw Data: "${data}"`);
      let payload;
      try {
        payload = JSON.parse(data);
        console.log('📦 [Scanner] Payload parsed:', payload);
      } catch (e) {
        console.error('❌ [Scanner] JSON Parse Error:', e);
        throw new Error('FORMAT_ERROR');
      }

      // Handle double-encoded QR: ticket_signature in DB is already JSON,
      // so the QR data might have sig as a nested JSON string
      let sid = String(payload.sid || '').trim();
      let uid = String(payload.uid || '').trim();
      let wid = String(payload.wid || '').trim();
      let sig = String(payload.sig || '').trim();

      // If sig looks like JSON (starts with {), it's the full ticket payload
      // Extract the actual base64 signature from inside
      if (sig.startsWith('{')) {
        try {
          const innerPayload = JSON.parse(sig);
          sid = sid || String(innerPayload.sid || '').trim();
          uid = uid || String(innerPayload.uid || '').trim();
          wid = wid || String(innerPayload.wid || '').trim();
          sig = String(innerPayload.sig || '').trim();
          console.log('📦 [Scanner] Extracted inner sig from nested JSON');
        } catch (e) {
          console.warn('⚠️ [Scanner] Failed to parse nested sig JSON');
        }
      }

      if (!sid || !wid || !uid || !sig) {
        console.warn('⚠️ [Scanner] Missing required fields in payload');
        throw new Error('FORMAT_ERROR');
      }

      // 4. XÁC THỰC CHỮ KÝ RSA (OFFLINE)
      console.log('🔐 [Scanner] Verifying RSA signature...');
      const isAuthentic = await verifyTicket({ sid, uid, wid, sig });
      if (!isAuthentic) {
        console.error('❌ [Scanner] RSA Signature Verification FAILED!');
        setScanStatus('error');
        setMessage('❌ CHỮ KÝ KHÔNG HỢP LỆ!');
        return;
      }

      console.log(`✅ [Scanner] Signature VALID. SV: ${sid}, WS: ${wid}`);

      // 4. KIỂM TRA PHÒNG
      if (wid !== currentWorkshop.id) {
        console.warn(`⚠️ [Scanner] Wrong Workshop. Expected: ${currentWorkshop.id}, Got: ${wid}`);
        setScanStatus('warning');
        const roomName = currentWorkshop.room;
        const displayRoom = (roomName.startsWith('Phòng') || roomName.startsWith('P.')) ? roomName : `Phòng ${roomName}`;
        setMessage(`⚠️ SAI WORKSHOP! (${displayRoom})`);
      }
      else {
        // 5. KIỂM TRA QUÉT TRÙNG (OFFLINE)
        const alreadyDone = await isStudentCheckedIn(sid, wid);
        if (alreadyDone) {
          console.warn(`⚠️ [Scanner] Student ${sid} already checked in!`);
          setScanStatus('warning');
          setMessage(`⚠️ VÉ ĐÃ SỬ DỤNG! (${sid})`);
        } else {
          // 6. LƯU LOCAL
          try {
            console.log(`💾 [Scanner] Saving check-in for ${sid}...`);
            await saveCheckinLocal({
              id: `${sid}_${Date.now()}`,
              student_id: sid,
              user_uuid: uid,
              workshop_id: wid,
              workshop_title: currentWorkshop.title,
              scanned_at: Date.now()
            });

            setScanStatus('success');
            setMessage(`✅ THÀNH CÔNG: ${sid}`);
            setCount(prev => prev + 1);
            onScanSuccess(payload);
          } catch (dbError: any) {
            console.error('❌ [Database Error]:', dbError.message);
            throw new Error('DATABASE_ERROR');
          }
        }
      }
    } catch (e: any) {
      console.error('🔥 [Scanner] Unexpected Error:', e.message);
      if (e.message === 'FORMAT_ERROR') {
        setScanStatus('error');
        setMessage('❌ MÃ QR KHÔNG HỢP LỆ');
      } else {
        setScanStatus('error');
        setMessage('❌ LỖI HỆ THỐNG (SQLITE)');
      }
    }

    // RESET ĐỂ QUÉT TIẾP SAU 1.5 GIÂY
    setTimeout(() => {
      setScanStatus('idle');
      setMessage('');
      isLocked.current = false;
    }, 1500);
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'success': return '#00C853'; // Xanh Neon
      case 'error': return '#FF1744';   // Đỏ Neon
      case 'warning': return '#FFAB00'; // Vàng Neon
      case 'processing': return '#2196F3'; // Xanh dương Loading
      default: return 'white';
    }
  };

  return (
    <View style={styles.container}>
      <View style={styles.banner}>
        <Text style={styles.bannerLabel}>Đang quét tại: {currentWorkshop.room}</Text>
        <Text style={styles.bannerTitle} numberOfLines={1}>{currentWorkshop.title}</Text>
      </View>

      <CameraView
        style={StyleSheet.absoluteFillObject}
        onBarcodeScanned={handleBarCodeScanned}
        barcodeScannerSettings={{ barcodeTypes: ['qr'] }}
      />

      <View style={styles.overlay}>
        <TouchableOpacity style={styles.closeBtn} onPress={onClose}>
          <X color="white" size={30} />
        </TouchableOpacity>

        {/* Khung ngắm Viewfinder */}
        <View style={[styles.viewfinder, (scanStatus !== 'idle' && scanStatus !== 'processing') && { borderColor: getStatusColor(scanStatus) }]}>
          <View style={styles.cornerTL} />
          <View style={styles.cornerTR} />
          <View style={styles.cornerBL} />
          <View style={styles.cornerBR} />
          {scanStatus === 'processing' && <ActivityIndicator size="large" color="white" style={styles.loader} />}
        </View>

        {/* Thông báo Toast */}
        {(scanStatus !== 'idle' && scanStatus !== 'processing') && (
          <View style={[styles.toast, { backgroundColor: getStatusColor(scanStatus) }]}>
            <Text style={styles.toastText}>{message}</Text>
          </View>
        )}

        <View style={styles.footer}>
          <View style={styles.counterBox}>
            <Text style={styles.counterLabel}>Đã quét</Text>
            <Text style={styles.counterValue}>{count}/60</Text>
          </View>
        </View>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: 'black' },
  center: { flex: 1, justifyContent: 'center', alignItems: 'center', padding: 20 },
  text: { color: '#64748B', textAlign: 'center', marginBottom: 20 },
  btn: { backgroundColor: '#1E40AF', padding: 15, borderRadius: 12 },
  btnText: { color: 'white', fontWeight: 'bold' },
  banner: {
    position: 'absolute', top: 0, left: 0, right: 0,
    backgroundColor: '#312E81', padding: 20, paddingTop: 50,
    zIndex: 10, alignItems: 'center'
  },
  bannerLabel: { color: '#BFDBFE', fontSize: 12, fontWeight: '500' },
  bannerTitle: { color: 'white', fontSize: 18, fontWeight: 'bold', marginTop: 4 },
  overlay: { flex: 1, backgroundColor: 'rgba(0,0,0,0.3)', justifyContent: 'center', alignItems: 'center' },
  closeBtn: { position: 'absolute', top: 120, right: 20, backgroundColor: 'rgba(0,0,0,0.5)', padding: 10, borderRadius: 25, zIndex: 30 },
  viewfinder: {
    width: width * 0.7, height: width * 0.7,
    borderWidth: 2, borderColor: 'rgba(255,255,255,0.5)',
    borderRadius: 24, position: 'relative', justifyContent: 'center', alignItems: 'center'
  },
  loader: { position: 'absolute' },
  cornerTL: { position: 'absolute', top: -2, left: -2, width: 40, height: 40, borderTopWidth: 5, borderLeftWidth: 5, borderColor: 'white', borderTopLeftRadius: 24 },
  cornerTR: { position: 'absolute', top: -2, right: -2, width: 40, height: 40, borderTopWidth: 5, borderRightWidth: 5, borderColor: 'white', borderTopRightRadius: 24 },
  cornerBL: { position: 'absolute', bottom: -2, left: -2, width: 40, height: 40, borderBottomWidth: 5, borderLeftWidth: 5, borderColor: 'white', borderBottomLeftRadius: 24 },
  cornerBR: { position: 'absolute', bottom: -2, right: -2, width: 40, height: 40, borderBottomWidth: 5, borderRightWidth: 5, borderColor: 'white', borderBottomRightRadius: 24 },
  toast: {
    position: 'absolute', top: '75%',
    paddingHorizontal: 24, paddingVertical: 12,
    borderRadius: 30, elevation: 5, zIndex: 40
  },
  toastText: { color: 'white', fontWeight: 'bold', fontSize: 16 },
  footer: { position: 'absolute', bottom: 50, left: 0, right: 0, alignItems: 'center' },
  counterBox: { backgroundColor: 'white', paddingHorizontal: 30, paddingVertical: 12, borderRadius: 20, alignItems: 'center' },
  counterLabel: { color: '#64748B', fontSize: 12 },
  counterValue: { color: '#1E293B', fontSize: 20, fontWeight: 'bold' },
});
