import { PostFeed } from '@/components/community/post-feed'
export default function Page() {
  return (
    <div className="mx-auto max-w-3xl">
      <h1 className="mb-6 text-3xl font-bold">Bài đăng doanh nghiệp</h1>
      <PostFeed fixedMode="mine" />
    </div>
  )
}
