'use client'
import Link from 'next/link'
import { useParams } from 'next/navigation'
import { useState } from 'react'
import { api } from '@/lib/api-client'
import { Button } from '@/components/ui/button'
import { toast } from 'sonner'
import { PostFeed } from './post-feed'
import { WorkshopPreview } from './workshop-preview'
import {
  type Company,
  type Workshop,
  CommunityShell,
  useResource,
  LoadState,
  errorText,
} from './shared'

export function CompaniesScreen() {
  const { data, loading, error, refresh } =
    useResource<Company[]>('/api/v1/companies')
  return (
    <CommunityShell
      title="Kết nối doanh nghiệp"
      description="Tìm hiểu đơn vị tổ chức và theo dõi những workshop bạn quan tâm."
    >
      <LoadState loading={loading} error={error} retry={() => void refresh()} />
      <div className="grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
        {data?.map((company) => (
          <Link
            key={company.id}
            href={`/companies/${company.slug}`}
            className="rounded-2xl border bg-white p-6 shadow-sm transition hover:border-indigo-300 hover:shadow-md"
          >
            <div className="flex h-14 w-14 items-center justify-center overflow-hidden rounded-xl bg-indigo-50 text-xl font-bold text-indigo-600">
              {company.logoUrl ? (
                <img
                  src={company.logoUrl}
                  alt={company.name}
                  className="h-full w-full object-cover"
                />
              ) : (
                company.name[0]
              )}
            </div>
            <h2 className="mt-4 text-lg font-bold">{company.name}</h2>
            <p className="mt-1 text-xs font-medium text-indigo-600">
              {company.industry || 'Đối tác UniHub'}
            </p>
            <p className="mt-3 line-clamp-3 text-sm leading-6 text-slate-500">
              {company.description ||
                'Khám phá các hoạt động và workshop của doanh nghiệp.'}
            </p>
            <p className="mt-5 text-xs text-slate-400">
              {company.followers} người theo dõi →
            </p>
          </Link>
        ))}
      </div>
      {data?.length === 0 && (
        <p className="py-12 text-center text-slate-500">
          Chưa có doanh nghiệp được duyệt.
        </p>
      )}
    </CommunityShell>
  )
}
function CompanyWorkshops({ id }: { id: string }) {
  const { data, loading, error, refresh } = useResource<Workshop[]>(
    `/api/v1/companies/${id}/workshops`
  )
  return (
    <div className="space-y-4">
      <LoadState loading={loading} error={error} retry={() => void refresh()} />
      {data?.map((w) => (
        <WorkshopPreview key={w.id} workshop={w} />
      ))}
      {data?.length === 0 && (
        <p className="text-sm text-slate-500">Chưa có workshop được công bố.</p>
      )}
    </div>
  )
}
export function CompanyScreen() {
  const { slug } = useParams<{ slug: string }>()
  const {
    data: company,
    loading,
    error,
    refresh,
  } = useResource<Company>(`/api/v1/companies/${slug}`)
  const [busy, setBusy] = useState(false)
  async function follow() {
    if (!company) return
    setBusy(true)
    try {
      if (company.following)
        await api.delete(`/api/v1/companies/${company.id}/follow`)
      else await api.put(`/api/v1/companies/${company.id}/follow`)
      await refresh()
    } catch (e) {
      toast.error(errorText(e))
    } finally {
      setBusy(false)
    }
  }
  return (
    <CommunityShell title={company?.name || 'Hồ sơ doanh nghiệp'}>
      <LoadState loading={loading} error={error} retry={() => void refresh()} />
      {company && (
        <>
          <section className="mb-8 overflow-hidden rounded-2xl border bg-white">
            {company.coverUrl && (
              <img
                src={company.coverUrl}
                alt="Ảnh bìa doanh nghiệp"
                className="h-48 w-full object-cover"
              />
            )}
            <div className="p-6">
              <div className="flex flex-wrap items-center justify-between gap-4">
                <div>
                  <p className="text-sm font-semibold text-indigo-600">
                    {company.industry} · {company.followers} người theo dõi
                  </p>
                  <h2 className="mt-2 text-2xl font-bold">{company.name}</h2>
                </div>
                <Button
                  disabled={busy}
                  variant={company.following ? 'outline' : 'default'}
                  onClick={() => void follow()}
                >
                  {company.following
                    ? 'Đang theo dõi'
                    : 'Theo dõi doanh nghiệp'}
                </Button>
              </div>
              <p className="mt-4 whitespace-pre-wrap text-sm leading-7 text-slate-600">
                {company.description}
              </p>
              <div className="mt-4 flex flex-wrap gap-4 text-sm text-slate-500">
                {company.address && <p>{company.address}</p>}
                {company.contact && <p>{company.contact}</p>}
                {company.website && (
                  <a
                    href={company.website}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-indigo-600"
                  >
                    Website ↗
                  </a>
                )}
              </div>
            </div>
          </section>
          <div className="grid gap-8 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
            <section>
              <h2 className="mb-4 text-xl font-bold">Bài viết & hoạt động</h2>
              <PostFeed companyId={company.id} />
            </section>
            <aside>
              <h2 className="mb-4 text-xl font-bold">Workshop</h2>
              <CompanyWorkshops id={company.id} />
            </aside>
          </div>
        </>
      )}
    </CommunityShell>
  )
}
