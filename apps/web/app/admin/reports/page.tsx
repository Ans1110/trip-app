import { AdminShell } from "@/components/admin/admin-shell";
import { ReportsView } from "@/components/admin/reports-view";

export default function AdminReportsPage() {
  return (
    <AdminShell title="Reports">
      <ReportsView />
    </AdminShell>
  );
}
