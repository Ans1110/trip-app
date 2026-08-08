import { AdminShell } from "@/components/admin/admin-shell";
import { UsersView } from "@/components/admin/users-view";

export default function AdminUsersPage() {
  return (
    <AdminShell title="Users">
      <UsersView />
    </AdminShell>
  );
}
