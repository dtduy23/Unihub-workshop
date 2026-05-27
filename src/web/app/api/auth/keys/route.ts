import { NextResponse } from 'next/server';
import { getPublicKey } from '@/lib/crypto';

/**
 * API cung cấp Public Key cho App Mobile để xác thực mã QR.
 * Endpoint: GET /api/auth/keys
 */
export async function GET() {
  try {
    const publicKey = getPublicKey();
    
    return NextResponse.json({
      publicKey,
      algorithm: 'Ed25519',
      issuedAt: new Date().toISOString()
    });
  } catch (error) {
    return NextResponse.json(
      { error: 'Internal Server Error' },
      { status: 500 }
    );
  }
}
