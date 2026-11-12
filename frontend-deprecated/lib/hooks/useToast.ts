"use client";

import { useState, useCallback } from "react";
import type { ToastMessage, ToastType } from "@/components/ui/Toast";

export function useToast() {
  const [toasts, setToasts] = useState<ToastMessage[]>([]);

  const addToast = useCallback(
    (type: ToastType, message: string, title?: string, duration = 4500) => {
      const id = `${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;
      setToasts((prev) => [...prev, { id, type, message, title, duration }]);
    },
    [],
  );

  const removeToast = useCallback((id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  }, []);

  const success = useCallback(
    (message: string, title = "Berhasil") => {
      addToast("success", message, title);
    },
    [addToast],
  );

  const error = useCallback(
    (message: string, title = "Gagal") => {
      addToast("error", message, title);
    },
    [addToast],
  );

  const warning = useCallback(
    (message: string, title = "Perhatian") => {
      addToast("warning", message, title);
    },
    [addToast],
  );

  const info = useCallback(
    (message: string, title = "Informasi") => {
      addToast("info", message, title);
    },
    [addToast],
  );

  return {
    toasts,
    removeToast,
    toast: {
      success,
      error,
      warning,
      info,
    },
  };
}
