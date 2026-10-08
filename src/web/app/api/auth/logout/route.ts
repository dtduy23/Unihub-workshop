import { NextRequest, NextResponse } from 'next/server'
export function POST(request: NextRequest) {
  const origin = request.headers.get('origin')
  let valid = request.headers.get('sec-fetch-site') !== 'cross-site'
  try {
    valid =
      valid && (!origin || new URL(origin).host === request.headers.get('host'))
  } catch {
    valid = false
  }
  if (!valid) return NextResponse.json({ success: false }, { status: 403 })
  const response = NextResponse.json({ success: true })
  response.cookies.delete('unihub_token')
  response.cookies.delete('unihub_session')
  return response
}
