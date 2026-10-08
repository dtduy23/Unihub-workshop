import { NextRequest, NextResponse } from 'next/server'

const backend = (
  process.env.API_INTERNAL_URL ||
  process.env.NEXT_PUBLIC_API_URL ||
  'http://localhost:8080'
).replace(/\/$/, '')
const authenticationPaths = new Set([
  '/api/v1/auth/login',
  '/api/v1/auth/forgot-password',
  '/api/v1/auth/reset-password',
])

async function forward(
  request: NextRequest,
  context: { params: Promise<{ path: string[] }> }
) {
  const { path } = await context.params
  const endpoint = `/${path.join('/')}`
  if (!endpoint.startsWith('/api/v1/') || path.some((part) => part === '..')) {
    return NextResponse.json(
      { success: false, error: 'Not found' },
      { status: 404 }
    )
  }
  if (!['GET', 'HEAD'].includes(request.method)) {
    const origin = request.headers.get('origin')
    let validOrigin = true
    try {
      validOrigin =
        !origin || new URL(origin).host === request.headers.get('host')
    } catch {
      validOrigin = false
    }
    if (
      !validOrigin ||
      request.headers.get('sec-fetch-site') === 'cross-site'
    ) {
      return NextResponse.json(
        { success: false, error: 'Invalid origin' },
        { status: 403 }
      )
    }
  }
  const token = request.cookies.get('unihub_token')?.value
  if (!token && !authenticationPaths.has(endpoint)) {
    return NextResponse.json(
      { success: false, error: 'Vui lòng đăng nhập' },
      { status: 401 }
    )
  }
  const headers = new Headers()
  const contentType = request.headers.get('content-type')
  if (contentType) headers.set('Content-Type', contentType)
  if (token) headers.set('Authorization', `Bearer ${token}`)
  try {
    const response = await fetch(
      `${backend}${endpoint}${request.nextUrl.search}`,
      {
        method: request.method,
        headers,
        body: ['GET', 'HEAD'].includes(request.method)
          ? undefined
          : await request.arrayBuffer(),
        cache: 'no-store',
        signal: AbortSignal.timeout(30_000),
      }
    )
    if (endpoint === '/api/v1/auth/login' && response.ok) {
      const result = await response.json()
      if (!result.data?.token)
        return NextResponse.json(
          { success: false, error: 'Invalid session' },
          { status: 502 }
        )
      const jwt = result.data.token
      result.data.token = ''
      const outgoing = NextResponse.json(result)
      outgoing.cookies.set('unihub_token', jwt, {
        httpOnly: true,
        sameSite: 'lax',
        secure: request.nextUrl.protocol === 'https:',
        path: '/',
        maxAge: 86400,
      })
      return outgoing
    }
    const outgoing = new NextResponse(response.body, {
      status: response.status,
    })
    for (const name of [
      'content-type',
      'content-disposition',
      'retry-after',
      'x-ratelimit-remaining',
      'x-content-type-options',
    ]) {
      const value = response.headers.get(name)
      if (value) outgoing.headers.set(name, value)
    }
    outgoing.headers.set('Cache-Control', 'private, no-store')
    if (
      response.status === 401 ||
      (endpoint === '/api/v1/auth/change-password' && response.ok)
    ) {
      outgoing.cookies.delete('unihub_token')
      outgoing.cookies.delete('unihub_session')
    }
    return outgoing
  } catch {
    return NextResponse.json(
      { success: false, error: 'Không thể kết nối máy chủ. Vui lòng thử lại.' },
      { status: 503 }
    )
  }
}

export const GET = forward
export const POST = forward
export const PUT = forward
export const PATCH = forward
export const DELETE = forward
export const HEAD = forward
