import { AdminShell } from "@/components/admin/admin-shell";
import { PostsView } from "@/components/admin/posts-view";

export default function AdminPostsPage() {
  return (
    <AdminShell title="Posts">
      <PostsView />
    </AdminShell>
  );
}
