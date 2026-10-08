import SiteNav from "@/components/SiteNav";
import AdminPanel from "@/components/AdminPanel";

export default function AdminPage() {
  return (
    <div className="min-h-screen">
      <SiteNav />
      <main className="container-page py-10">
        <AdminPanel />
      </main>
    </div>
  );
}
