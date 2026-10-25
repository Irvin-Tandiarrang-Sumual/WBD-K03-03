"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

const MENU = [
    { href: "/", label: "Home"},
    { href: "/databases", label: "Databases"},
    { href: "/admin", label: "Admin Control"},
]

export default function Navbar() {
    const pathname = usePathname();
    const isActive = (href) =>
    href === "/" ? pathname === "/" : pathname.startsWith(href);

  return (
    <header className="navbar">
      <div className="container">
        <Link href="/" className="brand" style={{ marginLeft: "1rem" }}>Wedebe</Link>

        <nav className="nav-links" aria-label="Main Navigation">
          {MENU.map((m) => (
            <Link
              key={m.href}
              href={m.href}
              className="nav-link"
              aria-current={isActive(m.href) ? "page" : undefined}
            >
              {m.label}
            </Link>
          ))}
        </nav>

      </div>
    </header>
  );
}