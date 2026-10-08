import Link from 'next/link'
import { Navbar } from '@/components/student/navbar'
export default function Layout({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-screen bg-slate-50">
      <Navbar />
      <div className="mx-auto max-w-6xl px-4 py-8 sm:px-6">
        <p className="text-xs font-bold uppercase tracking-widest text-indigo-600">
          Doanh nghiệp / UniHub
        </p>
        <nav className="my-5 flex flex-wrap gap-2">
          {[
            ['/business', 'Tổng quan'],
            ['/business/profile', 'Hồ sơ'],
            ['/business/workshops', 'Workshop'],
            ['/business/posts', 'Bài đăng'],
          ].map(([href, title]) => (
            <Link
              href={href}
              key={href}
              className="rounded-lg border bg-white px-4 py-2 text-sm font-medium text-slate-600 hover:border-indigo-300 hover:text-indigo-600"
            >
              {title}
            </Link>
          ))}
        </nav>
        {children}
      </div>
    </div>
  )
}
