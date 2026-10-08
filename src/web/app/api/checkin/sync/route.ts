import { NextRequest, NextResponse } from 'next/server'
export async function POST(request: NextRequest) {
  return NextResponse.redirect(new URL('/api/backend/api/v1/checkin/sync', request.url), 307)
}
