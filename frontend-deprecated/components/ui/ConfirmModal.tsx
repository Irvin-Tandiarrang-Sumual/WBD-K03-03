"use client";

import React, { useEffect } from "react";

export interface ConfirmModalProps {
  isOpen: boolean;
  title: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  variant?: "danger" | "warning" | "primary";
  isLoading?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export default function ConfirmModal({
  isOpen,
  title,
  message,
  confirmText = "Konfirmasi",
  cancelText = "Batal",
  variant = "danger",
  isLoading = false,
  onConfirm,
  onCancel,
}: ConfirmModalProps) {
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape" && isOpen && !isLoading) {
        onCancel();
      }
    }
    if (isOpen) {
      document.body.style.overflow = "hidden";
      window.addEventListener("keydown", handleKeyDown);
    }
    return () => {
      document.body.style.overflow = "unset";
      window.removeEventListener("keydown", handleKeyDown);
    };
  }, [isOpen, isLoading, onCancel]);

  if (!isOpen) return null;

  const isDanger = variant === "danger";

  return (
    <div
      style={{
        position: "fixed",
        inset: 0,
        zIndex: 9999,
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        backgroundColor: "rgba(0, 0, 0, 0.75)",
        backdropFilter: "blur(5px)",
        padding: "1rem",
        animation: "fadeIn 0.15s ease-out",
      }}
      onClick={(e) => {
        if (e.target === e.currentTarget && !isLoading) {
          onCancel();
        }
      }}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="confirm-modal-title"
        style={{
          width: "100%",
          maxWidth: "460px",
          backgroundColor: "#18181b",
          border: "1px solid #3f3f46",
          borderRadius: "12px",
          padding: "1.75rem",
          boxShadow: "0 20px 40px rgba(0, 0, 0, 0.6)",
          display: "flex",
          flexDirection: "column",
          gap: "1.25rem",
          animation: "scaleIn 0.2s ease-out",
        }}
      >
        <div style={{ display: "flex", gap: "1rem", alignItems: "flex-start" }}>
          <div
            style={{
              width: "42px",
              height: "42px",
              borderRadius: "50%",
              backgroundColor: isDanger
                ? "rgba(239, 68, 68, 0.15)"
                : "rgba(59, 130, 246, 0.15)",
              color: isDanger ? "#ef4444" : "#3b82f6",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              fontSize: "1.3rem",
              flexShrink: 0,
            }}
          >
            {isDanger ? "⚠️" : "ℹ️"}
          </div>

          <div style={{ flex: 1 }}>
            <h2
              id="confirm-modal-title"
              style={{
                fontSize: "1.15rem",
                fontWeight: 700,
                color: "#fafafa",
                margin: 0,
                lineHeight: 1.3,
              }}
            >
              {title}
            </h2>
            <p
              style={{
                fontSize: "0.875rem",
                color: "#a1a1aa",
                marginTop: "0.5rem",
                marginBottom: 0,
                lineHeight: 1.5,
                whiteSpace: "pre-line",
              }}
            >
              {message}
            </p>
          </div>
        </div>

        <div
          style={{
            display: "flex",
            justifyContent: "flex-end",
            gap: "0.75rem",
            marginTop: "0.5rem",
          }}
        >
          <button
            type="button"
            disabled={isLoading}
            onClick={onCancel}
            style={{
              padding: "0.55rem 1.1rem",
              borderRadius: "6px",
              backgroundColor: "transparent",
              border: "1px solid #3f3f46",
              color: "#e4e4e7",
              fontSize: "0.875rem",
              fontWeight: 500,
              cursor: isLoading ? "not-allowed" : "pointer",
              transition: "background 0.15s ease",
            }}
            onMouseOver={(e) => {
              if (!isLoading) e.currentTarget.style.backgroundColor = "#27272a";
            }}
            onMouseOut={(e) => {
              if (!isLoading)
                e.currentTarget.style.backgroundColor = "transparent";
            }}
          >
            {cancelText}
          </button>

          <button
            type="button"
            disabled={isLoading}
            onClick={onConfirm}
            style={{
              padding: "0.55rem 1.25rem",
              borderRadius: "6px",
              backgroundColor: isDanger ? "#dc2626" : "#2563eb",
              border: "none",
              color: "#ffffff",
              fontSize: "0.875rem",
              fontWeight: 600,
              cursor: isLoading ? "not-allowed" : "pointer",
              opacity: isLoading ? 0.7 : 1,
              display: "flex",
              alignItems: "center",
              gap: "0.5rem",
              transition: "background 0.15s ease",
            }}
            onMouseOver={(e) => {
              if (!isLoading)
                e.currentTarget.style.backgroundColor = isDanger
                  ? "#b91c1c"
                  : "#1d4ed8";
            }}
            onMouseOut={(e) => {
              if (!isLoading)
                e.currentTarget.style.backgroundColor = isDanger
                  ? "#dc2626"
                  : "#2563eb";
            }}
          >
            {isLoading ? (
              <>
                <span
                  style={{
                    width: "14px",
                    height: "14px",
                    border: "2px solid #ffffff",
                    borderTop: "2px solid transparent",
                    borderRadius: "50%",
                    display: "inline-block",
                    animation: "spin 0.6s linear infinite",
                  }}
                />
                Memproses...
              </>
            ) : (
              confirmText
            )}
          </button>
        </div>
      </div>

      <style>{`
        @keyframes fadeIn {
          from { opacity: 0; }
          to { opacity: 1; }
        }
        @keyframes scaleIn {
          from { opacity: 0; transform: scale(0.95); }
          to { opacity: 1; transform: scale(1); }
        }
        @keyframes spin {
          from { transform: rotate(0deg); }
          to { transform: rotate(360deg); }
        }
      `}</style>
    </div>
  );
}
