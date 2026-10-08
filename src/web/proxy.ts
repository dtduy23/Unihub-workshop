import { NextResponse, type NextRequest } from 'next/server'

const backend = (
  process.env.API_INTERNAL_URL ||
  process.env.NEXT_PUBLIC_API_URL ||
  'http://localhost:8080'
).replace(/\/$/, '')
const roleHome: Record<string, string> = {
  STUDENT: '/',
  STAFF: '/staff/checkin',
  BUSINESS: '/business',
  ADMIN: '/admin',
}

export async function proxy(request: NextRequest) {
  const path = request.nextUrl.pathname
  // The API proxy authenticates each request independently; login and recovery need no session.
  if (path.startsWith('/api/backend/') || path === '/api/auth/logout')
    return NextResponse.next()
  const publicPage = path === '/login' || path === '/reset-password'
  const token = request.cookies.get('unihub_token')?.value
  const login = () => {
    const response = NextResponse.redirect(new URL('/login', request.url))
    response.cookies.delete('unihub_token')
    response.cookies.delete('unihub_session')
    return response
  }
  if (!token) return publicPage ? NextResponse.next() : login()
  let role: string
  try {
    const result = await fetch(`${backend}/api/v1/auth/me`, {
      headers: { Authorization: `Bearer ${token}` },
      cache: 'no-store',
      signal: AbortSignal.timeout(5000),
    })
    if (result.status >= 500)
      return publicPage
        ? NextResponse.next()
        : new NextResponse(
            'Máy chủ tạm thời không khả dụng. Vui lòng tải lại trang.',
            { status: 503 }
          )
    if (!result.ok) return publicPage ? NextResponse.next() : login()
    const data = await result.json()
    role = data.data?.role
    if (!roleHome[role]) return login()
  } catch {
    if (publicPage) return NextResponse.next()
    return new NextResponse(
      'Máy chủ tạm thời không khả dụng. Vui lòng tải lại trang.',
      { status: 503 }
    )
  }
  if (path === '/login')
    return NextResponse.redirect(new URL(roleHome[role], request.url))
  if (path.startsWith('/admin') && role !== 'ADMIN')
    return NextResponse.redirect(new URL(roleHome[role], request.url))
  if (path.startsWith('/business') && role !== 'BUSINESS')
    return NextResponse.redirect(new URL(roleHome[role], request.url))
  if (path.startsWith('/staff') && role !== 'STAFF' && role !== 'ADMIN')
    return NextResponse.redirect(new URL(roleHome[role], request.url))
  return NextResponse.next()
}
export const config = {
  matcher: [
    '/((?!_next/static|_next/image|favicon.ico|icon.svg|icon-light-32x32.png|icon-dark-32x32.png|apple-icon.png|placeholder|maps/|images/).*)',
  ],
}
