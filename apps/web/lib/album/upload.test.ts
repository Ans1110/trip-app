import { describe, expect, mock, test } from "bun:test";

import type { AlbumPhoto } from "../apis/album-api";
import {
  runUploadBatch,
  UploadError,
  uploadOnePhoto,
  type BatchEvent,
  type UploadDeps,
  type UploadTicket,
} from "./upload";

// ---- fixtures ----

const makeFile = (name: string, type: string, size: number): File => {
  // Buffer size doesn't matter — we never read it in tests, only .size.
  const blob = new Blob([new Uint8Array(size)], { type });
  return new File([blob], name, { type, lastModified: 0 });
};

const stubPhoto = (id: string): AlbumPhoto => ({
  id,
  trip_id: "trip-1",
  media_id: `media-${id}`,
  added_by: "user-1",
  taken_at: "2026-01-01T00:00:00Z",
  caption: "",
  created_at: "2026-01-01T00:00:00Z",
  original_url: `https://s3/${id}/original`,
  thumb_small_url: `https://s3/${id}/small`,
});

const makeDeps = (overrides: Partial<UploadDeps> = {}): UploadDeps => ({
  isHeic: () => false,
  convertHeicToJpeg: async (f) => f,
  initUpload: async () => ({
    slots: [
      {
        upload_id: "u-1",
        upload_url: "https://s3/put",
        method: "PUT",
        headers: {},
        expires_at: "2026-01-01T00:15:00Z",
      },
    ],
  }),
  putBinary: async () => undefined,
  completePhoto: async () => stubPhoto("photo-1"),
  ...overrides,
});

// ---- uploadOnePhoto ----

describe("uploadOnePhoto", () => {
  test("happy path drives phases in order and returns the photo", async () => {
    const phases: string[] = [];
    const file = makeFile("a.jpg", "image/jpeg", 1024);
    const photo = await uploadOnePhoto(file, "trip-1", makeDeps(), {
      onPhase: (p) => phases.push(p),
    });
    expect(photo.id).toBe("photo-1");
    expect(phases).toEqual(["initiating", "uploading", "completing", "done"]);
  });

  test("converts when isHeic returns true", async () => {
    const convert = mock(async (f: File) =>
      new File([f], "converted.jpg", { type: "image/jpeg" }),
    );
    let sentMime = "";
    const initUpload = mock(async (_trip: string, input: { items: { mime: string }[] }) => {
      sentMime = input.items[0].mime;
      return { slots: [{ upload_id: "u", upload_url: "u", method: "PUT", expires_at: "" }] };
    });
    const file = makeFile("IMG.HEIC", "image/heic", 2048);
    await uploadOnePhoto(file, "trip-1", makeDeps({ isHeic: () => true, convertHeicToJpeg: convert, initUpload }));
    expect(convert).toHaveBeenCalledTimes(1);
    // initUpload must have seen the converted mime, not the original heic.
    expect(sentMime).toBe("image/jpeg");
  });

  test("HEIC conversion failure throws UploadError with phase='converting'", async () => {
    const deps = makeDeps({
      isHeic: () => true,
      convertHeicToJpeg: async () => {
        throw new Error("libheif decode failed");
      },
    });
    const file = makeFile("IMG.HEIC", "image/heic", 2048);
    let caught: unknown;
    try {
      await uploadOnePhoto(file, "trip-1", deps);
    } catch (err) {
      caught = err;
    }
    expect(caught).toBeInstanceOf(UploadError);
    expect((caught as UploadError).phase).toBe("converting");
  });

  test("rejects unsupported mime before init-upload is called", async () => {
    const init = mock(async () => ({ slots: [] }));
    const deps = makeDeps({ initUpload: init });
    const file = makeFile("doc.pdf", "application/pdf", 1024);
    let caught: unknown;
    try {
      await uploadOnePhoto(file, "trip-1", deps);
    } catch (err) {
      caught = err;
    }
    expect(caught).toBeInstanceOf(UploadError);
    expect((caught as UploadError).phase).toBe("queued");
    expect(init).not.toHaveBeenCalled();
  });

  test("rejects oversized files before init-upload", async () => {
    const init = mock(async () => ({ slots: [] }));
    const deps = makeDeps({ initUpload: init });
    const file = makeFile("huge.jpg", "image/jpeg", 60 * 1024 * 1024);
    let caught: unknown;
    try {
      await uploadOnePhoto(file, "trip-1", deps);
    } catch (err) {
      caught = err;
    }
    expect(caught).toBeInstanceOf(UploadError);
    expect((caught as UploadError).phase).toBe("queued");
    expect(init).not.toHaveBeenCalled();
  });

  test("bubbles putBinary failure with phase='uploading'", async () => {
    const deps = makeDeps({
      putBinary: async () => {
        throw new Error("Network error during upload");
      },
    });
    const file = makeFile("a.jpg", "image/jpeg", 1024);
    let caught: unknown;
    try {
      await uploadOnePhoto(file, "trip-1", deps);
    } catch (err) {
      caught = err;
    }
    expect(caught).toBeInstanceOf(UploadError);
    expect((caught as UploadError).phase).toBe("uploading");
    expect((caught as UploadError).message).toBe("Network error during upload");
  });

  test("bubbles completePhoto failure with phase='completing'", async () => {
    const deps = makeDeps({
      completePhoto: async () => {
        throw new Error("finalise blew up");
      },
    });
    const file = makeFile("a.jpg", "image/jpeg", 1024);
    let caught: unknown;
    try {
      await uploadOnePhoto(file, "trip-1", deps);
    } catch (err) {
      caught = err;
    }
    expect(caught).toBeInstanceOf(UploadError);
    expect((caught as UploadError).phase).toBe("completing");
  });

  test("handles empty slots response", async () => {
    const deps = makeDeps({ initUpload: async () => ({ slots: [] }) });
    const file = makeFile("a.jpg", "image/jpeg", 1024);
    let caught: unknown;
    try {
      await uploadOnePhoto(file, "trip-1", deps);
    } catch (err) {
      caught = err;
    }
    expect(caught).toBeInstanceOf(UploadError);
    expect((caught as UploadError).phase).toBe("initiating");
  });
});

// ---- runUploadBatch ----

describe("runUploadBatch", () => {
  const mkTicket = (id: string, file: File): UploadTicket => ({
    id,
    originalFile: file,
  });

  test("returns per-file results with successes and failures isolated", async () => {
    let call = 0;
    const deps = makeDeps({
      putBinary: async () => {
        call++;
        if (call === 2) throw new Error("second upload blew up");
      },
      completePhoto: async () => stubPhoto(`ok-${call}`),
    });
    const tickets = [
      mkTicket("a", makeFile("a.jpg", "image/jpeg", 100)),
      mkTicket("b", makeFile("b.jpg", "image/jpeg", 100)),
      mkTicket("c", makeFile("c.jpg", "image/jpeg", 100)),
    ];
    const events: BatchEvent[] = [];
    const results = await runUploadBatch(
      tickets,
      "trip-1",
      deps,
      (e) => events.push(e),
      { concurrency: 1 },
    );

    expect(results.length).toBe(3);
    const outcomes = results.map((r) => r.outcome.status);
    expect(outcomes.filter((s) => s === "success").length).toBe(2);
    expect(outcomes.filter((s) => s === "error").length).toBe(1);

    // The failed row must have flipped to phase 'error' so the UI can offer retry.
    const errorPhases = events.filter(
      (e) => e.kind === "phase" && e.phase === "error",
    );
    expect(errorPhases.length).toBe(1);
    expect(errorPhases[0].ticketId).toBe("b");
  });

  test("HEIC convert failure on one file does not fail the batch", async () => {
    const deps = makeDeps({
      isHeic: (f) => f.name.toLowerCase().endsWith(".heic"),
      convertHeicToJpeg: async (f) => {
        if (f.name === "bad.heic") throw new Error("decode failed");
        return new File([f], "ok.jpg", { type: "image/jpeg" });
      },
    });
    const tickets = [
      mkTicket("good", makeFile("good.jpg", "image/jpeg", 100)),
      mkTicket("bad", makeFile("bad.heic", "image/heic", 100)),
    ];
    const results = await runUploadBatch(tickets, "trip-1", deps, () => {}, {
      concurrency: 2,
    });
    const byId = Object.fromEntries(results.map((r) => [r.ticketId, r.outcome]));
    expect(byId.good.status).toBe("success");
    expect(byId.bad.status).toBe("error");
    if (byId.bad.status === "error") {
      expect(byId.bad.error.phase).toBe("converting");
    }
  });

  test("retry replays a single ticket after a transient failure", async () => {
    let attempt = 0;
    const deps = makeDeps({
      putBinary: async () => {
        attempt++;
        if (attempt === 1) throw new Error("flaky");
      },
    });
    const ticket = mkTicket("x", makeFile("a.jpg", "image/jpeg", 100));

    const first = await runUploadBatch([ticket], "trip-1", deps, () => {});
    expect(first[0].outcome.status).toBe("error");

    // Retry = call runUploadBatch again with the same ticket.
    const second = await runUploadBatch([ticket], "trip-1", deps, () => {});
    expect(second[0].outcome.status).toBe("success");
  });

  test("respects concurrency: never more than N in-flight uploads", async () => {
    let inFlight = 0;
    let peak = 0;
    const deps = makeDeps({
      putBinary: async () => {
        inFlight++;
        peak = Math.max(peak, inFlight);
        await new Promise((r) => setTimeout(r, 5));
        inFlight--;
      },
    });
    const tickets = Array.from({ length: 6 }, (_, i) =>
      mkTicket(`t${i}`, makeFile(`f${i}.jpg`, "image/jpeg", 100)),
    );
    await runUploadBatch(tickets, "trip-1", deps, () => {}, { concurrency: 2 });
    expect(peak).toBeLessThanOrEqual(2);
  });
});
