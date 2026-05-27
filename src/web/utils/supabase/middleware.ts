import { createServerClient, type CookieOptions } from '@supabase/ssr'
import { NextResponse, type NextRequest } from 'next/server'

export async function updateSession(request: NextRequest) {
  let response = NextResponse.next({
    request: {
      headers: request.headers,
    },
  })

  const supabase = createServerClient(
    process.env.NEXT_PUBLIC_SUPABASE_URL!,
    process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY!,
    {
      cookies: {
        get(name: string) {
          return request.cookies.get(name)?.value
        },
        set(name: string, value: string, options: CookieOptions) {
          request.cookies.set({
            name,
            value,
            ...options,
          })
          response = NextResponse.next({
            request: {
              headers: request.headers,
            },
          })
          response.cookies.set({
            name,
            value,
            ...options,
          })
        },
        remove(name: string, options: CookieOptions) {
          request.cookies.set({
            name,
            value: '',
            ...options,
          })
          response = NextResponse.next({
            request: {
              headers: request.headers,
            },
          })
          response.cookies.set({
            name,
            value: '',
            ...options,
          })
        },
      },
    }
  )

  const { data: { user } } = await supabase.auth.getUser()

  // 1. Nếu chưa đăng nhập và không phải đang ở trang /login -> Redirect về /login
  if (!user && !request.nextUrl.pathname.startsWith('/login')) {
    const url = request.nextUrl.clone()
    url.pathname = '/login'
    return NextResponse.redirect(url)
  }

  // 2. Nếu đã đăng nhập và đang ở trang /login -> Redirect về trang chủ tương ứng role
  if (user && request.nextUrl.pathname.startsWith('/login')) {
    const role = user.user_metadata?.role || 'STUDENT'
    const url = request.nextUrl.clone()
    url.pathname = role === 'ORGANIZER' ? '/admin' : '/'
    return NextResponse.redirect(url)
  }

  // 3. Bảo vệ riêng cho /admin
  if (user && request.nextUrl.pathname.startsWith('/admin')) {
    const role = user.user_metadata?.role || 'STUDENT'
    if (role !== 'ORGANIZER') {
      const url = request.nextUrl.clone()
      url.pathname = '/'
      return NextResponse.redirect(url)
    }
  }

  return response
}
