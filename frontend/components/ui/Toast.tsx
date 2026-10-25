"use client";

import React, { useEffect } from "react";

export type ToastType = "success" | "error" | "info" | "warning";

export interface ToastMessage {
  id: string;
  type: ToastType;
  title?: string;
  message: string;
  duration?: number;
}

interface ToastProps {
  toasts: ToastMessage[];
  onDismiss: (id: string) => void;
}

export default function ToastContainer({ toasts, onDismiss }: ToastProps) {
  if (toasts.length === 0) return null;

  return (
    <div
      style={{
        position: "fixed",
        top: "1.5rem",
        right: "1.5rem",
        zIndex: 10000,
        display: "flex",
        flexDirection: "column",
        gap: "0.75rem",
        maxWidth: "420px",
        width: "calc(100vw - 3rem)",
        pointerEvents: "none",
      }}
    >
      {toasts.map((toast) => (
        <ToastItem key={toast.id} toast={toast} onDismiss={onDismiss} />
      ))}
    </div>
  );
}

function ToastItem({
  toast,
  onDismiss,
}: {
  toast: ToastMessage;
  onDismiss: (id: string) => void;
}) {
  const { id, type, title, message, duration = 4500 } = toast;

  useEffect(() => {
    if (duration > 0) {
      const timer = setTimeout(() => {
        onDismiss(id);
      }, duration);
      return () => clearTimeout(timer);
    }
  }, [id, duration, onDismiss]);

  const config = {
    success: {
      icon: "✓",
      bgColor: "#0d2818",
      borderColor: "rgba(34, 197, 94, 0.4)",
      textColor: "#86efac",
      iconBg: "#15803d",
      defaultTitle: "Berhasil",
    },
    error: {
      icon: "✕",
      bgColor: "#2a0808",
      borderColor: "rgba(239, 68, 68, 0.4)",
      textColor: "#fca5a5",
      iconBg: "#b91c1c",
      defaultTitle: "Terjadi Kesalahan",
    },
    warning: {
      icon: "⚠️",
      bgColor: "#291c06",
      borderColor: "rgba(234, 179, 8, 0.4)",
      textColor: "#fde047",
      iconBg: "#a16207",
      defaultTitle: "Perhatian",
    },
    info: {
      icon: "ℹ️",
      bgColor: "#0c1f38",
      borderColor: "rgba(59, 130, 246, 0.4)",
      textColor: "#93c5fd",
      iconBg: "#1d4ed8",
      defaultTitle: "Informasi",
    },
  }[type];

  return (
    <div
      role="alert"
      style={{
        pointerEvents: "auto",
        backgroundColor: config.bgColor,
        border: `1px solid ${config.borderColor}`,
        borderRadius: "10px",
        padding: "0.85rem 1rem",
        boxShadow: "0 10px 25px rgba(0, 0, 0, 0.6)",
        display: "flex",
        alignItems: "flex-start",
        gap: "0.75rem",
        animation: "slideIn 0.25s ease-out",
        color: "#fafafa",
        fontSize: "0.875rem",
      }}
    >
      <div
        style={{
          width: "24px",
          height: "24px",
          borderRadius: "50%",
          backgroundColor: config.iconBg,
          color: "#ffffff",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          fontSize: "0.75rem",
          fontWeight: 800,
          flexShrink: 0,
          marginTop: "1px",
        }}
      >
        {config.icon}
      </div>

      <div style={{ flex: 1, minWidth: 0 }}>
        <strong
          style={{
            display: "block",
            color: config.textColor,
            fontSize: "0.875rem",
            marginBottom: "0.2rem",
          }}
        >
          {title || config.defaultTitle}
        </strong>
        <p
          style={{
            margin: 0,
            color: "#e4e4e7",
            lineHeight: 1.45,
            wordBreak: "break-word",
            fontSize: "0.85rem",
          }}
        >
          {message}
        </p>
      </div>

      <button
        type="button"
        onClick={() => onDismiss(id)}
        aria-label="Tutup notifikasi"
        style={{
          background: "transparent",
          border: "none",
          color: "#a1a1aa",
          cursor: "pointer",
          fontSize: "1rem",
          padding: "2px 6px",
          borderRadius: "4px",
          lineHeight: 1,
        }}
        onMouseOver={(e) => (e.currentTarget.style.color = "#ffffff")}
        onMouseOut={(e) => (e.currentTarget.style.color = "#a1a1aa")}
      >
        ✕
      </button>

      <style>{`
        @keyframes slideIn {
          from { transform: translateX(30px); opacity: 0; }
          to { transform: translateX(0); opacity: 1; }
        }
      `}</style>
    </div>
  );
}
