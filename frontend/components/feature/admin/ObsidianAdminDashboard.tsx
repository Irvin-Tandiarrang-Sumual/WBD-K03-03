import Link from "next/link";

const adminModules = [
  {
    icon: "🔌",
    title: "Database Connections",
    text: "Daftarkan koneksi database target baru (PostgreSQL / MySQL) dan kelola koneksi yang aktif.",
    href: "/admin/connections",
    action: "Kelola Connections →",
  },
  {
    icon: "👥",
    title: "Access Groups",
    text: "Atur grup izin akses ke database (View Only atau View & Edit) serta tetapkan user ke dalam grup.",
    href: "/admin/groups",
    action: "Kelola Groups →",
  },
  {
    icon: "👤",
    title: "User Management",
    text: "Buat pengguna baru, lihat daftar pengguna platform, dan perbarui peran (Role Admin / User).",
    href: "/admin/users",
    action: "Kelola Users →",
  },
];

export default function AdminDashboard() {
  return (
    <div
      style={{
        maxWidth: "1000px",
        margin: "0 auto",
        padding: "2.5rem 1.5rem",
        display: "flex",
        flexDirection: "column",
        gap: "2rem",
      }}
    >
      <header
        style={{
          borderBottom: "1px solid #27272a",
          paddingBottom: "1.5rem",
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          flexWrap: "wrap",
          gap: "1rem",
        }}
      >
        <div>
          <div
            style={{
              display: "inline-flex",
              alignItems: "center",
              gap: "0.4rem",
              backgroundColor: "rgba(124, 58, 237, 0.15)",
              border: "1px solid rgba(124, 58, 237, 0.4)",
              color: "#c084fc",
              padding: "2px 10px",
              borderRadius: "12px",
              fontSize: "0.75rem",
              fontWeight: 700,
              textTransform: "uppercase",
              marginBottom: "0.5rem",
            }}
          >
            🛡️ Admin Area
          </div>
          <h1 style={{ fontSize: "1.85rem", fontWeight: 800, color: "#fafafa" }}>
            Admin Control Center
          </h1>
          <p style={{ color: "#a1a1aa", fontSize: "0.95rem", marginTop: "0.25rem" }}>
            Pusat konfigurasi sistem, database eksternal, grup akses, dan manajemen pengguna.
          </p>
        </div>
      </header>

      <div
        style={{
          display: "grid",
          gridTemplateColumns: "repeat(auto-fit, minmax(280px, 1fr))",
          gap: "1.5rem",
        }}
      >
        {adminModules.map((item) => (
          <article
            key={item.href}
            style={{
              backgroundColor: "#121215",
              border: "1px solid #27272a",
              borderRadius: "12px",
              padding: "1.75rem",
              display: "flex",
              flexDirection: "column",
              gap: "1rem",
              transition: "transform 0.15s ease, border-color 0.15s ease",
            }}
          >
            <div style={{ fontSize: "2rem" }}>{item.icon}</div>
            <div>
              <h2 style={{ fontSize: "1.2rem", fontWeight: 700, color: "#fafafa", marginBottom: "0.35rem" }}>
                {item.title}
              </h2>
              <p style={{ color: "#a1a1aa", fontSize: "0.875rem", lineHeight: 1.5 }}>
                {item.text}
              </p>
            </div>
            <Link
              href={item.href}
              style={{
                marginTop: "auto",
                paddingTop: "0.5rem",
                color: "#60a5fa",
                fontWeight: 600,
                fontSize: "0.9rem",
              }}
            >
              {item.action}
            </Link>
          </article>
        ))}
      </div>
    </div>
  );
}
