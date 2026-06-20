"use client";

import { useEffect, type ReactNode } from "react";
import { X } from "lucide-react";

export function Modal({
  title,
  onClose,
  children,
}: {
  title: string;
  onClose: () => void;
  children: ReactNode;
}) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  return (
    <div className="fixed inset-0 z-50">
      <button
        type="button"
        aria-label="Close"
        onClick={onClose}
        className="absolute inset-0 cursor-default"
        style={{ backgroundColor: "rgba(0,0,0,0.55)" }}
      />
      <div className="absolute inset-0 flex items-center justify-center px-4 py-8">
        <div
          className="w-full max-w-md rounded-2xl overflow-hidden"
          style={{
            backgroundColor: "#0B100D",
            border: "1px solid #1F2A24",
            boxShadow: "0 24px 64px rgba(0,0,0,0.5)",
          }}
        >
          <div
            className="flex items-center justify-between px-6 py-4 border-b"
            style={{ borderColor: "#1F2A24" }}
          >
            <h2
              className="text-lg tracking-tight"
              style={{
                fontFamily: "var(--font-display, Georgia, serif)",
                color: "#ECEFEA",
              }}
            >
              {title}
            </h2>
            <button
              type="button"
              onClick={onClose}
              aria-label="Close"
              className="inline-flex items-center justify-center size-8 rounded-full hover:bg-white/5"
              style={{ color: "#8B9A8E" }}
            >
              <X className="size-4" />
            </button>
          </div>
          <div className="px-6 py-5">{children}</div>
        </div>
      </div>
    </div>
  );
}
