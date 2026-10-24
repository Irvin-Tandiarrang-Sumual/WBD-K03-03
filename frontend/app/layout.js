import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";
import Navbar from "@/components/layout/Navbar";
import {Suspense} from "react";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata = {
  title: {
    default: "Wedebe",
    template: "%s | Wedebe",
  },
  description: "Database Management Platform",
};

export default function RootLayout({ children }) {
  return (
    <html lang="en" className={`${geistSans.variable} ${geistMono.variable}`}>
      <body>
        <Suspense fallback={<div style={{ height: 60 }} />}>
          <Navbar />
        </Suspense>
        <main className="container page"> {children}</main>
      </body>
    </html>
  );
}