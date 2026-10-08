import { NextRequest, NextResponse } from 'next/server'
export async function GET(request: NextRequest) {
  return NextResponse.redirect(new URL('/api/backend/api/v1/auth/public-key', request.url), 307)
}
