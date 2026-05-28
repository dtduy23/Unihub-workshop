import { NextResponse, type NextRequest } from 'next/server'

/**
 * Middleware bảo vệ route — đọc session từ Cookie.
 *
 * Role mapping (từ Go Backend):
 * - ORGANIZER → truy cập /admin
 * - STAFF     → truy cập /admin (chỉ check-in)
 * - STUDENT   → truy cập / (Student Portal)
 *
 * Tuân thủ: auth.md mục 3.2, agent.md mục 10.4
 */
export function middleware(request: NextRequest) {
  const sessionCookie = request.cookies.get('unihub_session')?.value
  const tokenCookie = request.cookies.get('unihub_token')?.value
  const pathname = request.nextUrl.pathname

  // Các route public không cần kiểm tra auth
  const isPublicRoute =
    pathname.startsWith('/login') ||
    pathname.startsWith('/api') ||
    pathname.startsWith('/_next') ||
    pathname === '/favicon.ico'

  // 1. Route công khai — cho đi qua
  if (isPublicRoute) {
    // Nếu đã đăng nhập mà đang ở trang login → redirect về trang chủ
    if (sessionCookie && pathname.startsWith('/login')) {
      try {
        const user = JSON.parse(decodeURIComponent(sessionCookie))
        const role = user.role?.toUpperCase()
        let dest = '/'
        if (role === 'ADMIN') dest = '/admin'
        else if (role === 'STAFF') dest = '/staff/checkin'
        return NextResponse.redirect(new URL(dest, request.url))
      } catch {
        // Cookie hỏng → xóa và cho vào login
        const res = NextResponse.redirect(new URL('/login', request.url))
        res.cookies.delete('unihub_session')
        res.cookies.delete('unihub_token')
        return res
      }
    }
    return NextResponse.next()
  }

  // 2. Chưa đăng nhập → chuyển về login
  if (!sessionCookie || !tokenCookie) {
    return NextResponse.redirect(new URL('/login', request.url))
  }

  // 3. Bảo vệ /admin và /staff — chỉ ADMIN hoặc STAFF mới vào được
  if (pathname.startsWith('/admin') || pathname.startsWith('/staff')) {
    try {
      const user = JSON.parse(decodeURIComponent(sessionCookie))
      const role = user.role?.toUpperCase()
      if (role !== 'ADMIN' && role !== 'STAFF') {
        return NextResponse.redirect(new URL('/', request.url))
      }
    } catch {
      const res = NextResponse.redirect(new URL('/login', request.url))
      res.cookies.delete('unihub_session')
      res.cookies.delete('unihub_token')
      return res
    }
  }

  return NextResponse.next()
}

export const config = {
  matcher: [
    '/((?!_next/static|_next/image|favicon.ico|.*\\.(?:svg|png|jpg|jpeg|gif|webp)$).*)',
  ],
}
