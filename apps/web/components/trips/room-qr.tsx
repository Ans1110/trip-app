"use client";

import { useEffect, useRef, useState, useSyncExternalStore } from "react";
import QRCode from "qrcode";
import { Download, Share2 } from "lucide-react";

const subscribeOrigin = () => () => {};
const readOrigin = () => window.location.origin;
const readServerOrigin = () => "";

export function RoomQr({ code }: { code: string }) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const origin = useSyncExternalStore(
    subscribeOrigin,
    readOrigin,
    readServerOrigin,
  );
  const [shareState, setShareState] = useState<"idle" | "copied" | "sharing">(
    "idle",
  );

  const url = origin ? `${origin}/rooms/${code}` : "";

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas || !url) return;
    QRCode.toCanvas(canvas, url, {
      width: 220,
      margin: 1,
      color: { dark: "#0B100D", light: "#ECEFEA" },
    }).catch(() => {
      // ignore render failure — code text is still shown
    });
  }, [url]);

  const handleDownload = () => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const link = document.createElement("a");
    link.download = `trip-${code}.png`;
    link.href = canvas.toDataURL("image/png");
    link.click();
  };

  const canvasToPngFile = (): Promise<File | null> =>
    new Promise((resolve) => {
      const canvas = canvasRef.current;
      if (!canvas) return resolve(null);
      canvas.toBlob((blob) => {
        if (!blob) return resolve(null);
        resolve(new File([blob], `trip-${code}.png`, { type: "image/png" }));
      }, "image/png");
    });

  const handleShare = async () => {
    if (!url || shareState === "sharing") return;
    setShareState("sharing");
    const payload = {
      title: "Join my trip",
      text: `Join my trip on TripCraft (code ${code})`,
      url,
    };
    const copyFallback = async () => {
      try {
        await navigator.clipboard.writeText(url);
        setShareState("copied");
        setTimeout(() => setShareState("idle"), 1500);
      } catch {
        setShareState("idle");
      }
    };

    const canShareApi = typeof navigator.share === "function";
    if (!canShareApi) {
      await copyFallback();
      return;
    }
    try {
      const file = await canvasToPngFile();
      const withFile =
        file && navigator.canShare?.({ files: [file] })
          ? { ...payload, files: [file] }
          : payload;
      await navigator.share(withFile);
      setShareState("idle");
    } catch (err) {
      if ((err as DOMException)?.name === "AbortError") {
        setShareState("idle");
        return;
      }
      await copyFallback();
    }
  };

  return (
    <div className="flex flex-col items-center gap-3">
      <div className="rounded-xl p-3" style={{ backgroundColor: "#ECEFEA" }}>
        <canvas ref={canvasRef} aria-label={`QR code for trip ${code}`} />
      </div>

      <div className="flex items-center gap-2">
        <button
          type="button"
          onClick={handleShare}
          disabled={!url || shareState === "sharing"}
          className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-full hover:bg-white/5 disabled:opacity-60"
          style={{ color: "#ECEFEA", border: "1px solid #1F2A24" }}
        >
          <Share2 className="size-3.5" />
          {shareState === "copied" ? "Link copied!" : "Share"}
        </button>
        <button
          type="button"
          onClick={handleDownload}
          className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-full hover:bg-white/5"
          style={{ color: "#ECEFEA", border: "1px solid #1F2A24" }}
        >
          <Download className="size-3.5" />
          Download
        </button>
      </div>
    </div>
  );
}
