import { describe, expect, test } from "bun:test";
import { QueryClient } from "@tanstack/react-query";

import { albumKeys, invalidateAlbumForTrip } from "./album-hooks";

// invalidateAlbumForTrip is the single seam between the realtime dispatcher
// and TanStack — verifying it here means the switch in realtime-hooks
// (ALBUM_PHOTO_UPLOADED / ALBUM_PHOTO_UPDATED / etc.) has a trustworthy
// invalidation target. Realtime state contract: no local state mutation,
// only refetch, so an invalidated query is enough.

const seed = (qc: QueryClient, tripId: string) =>
  qc.setQueryData(albumKeys.photos(tripId), [{ id: "p1" }]);

describe("invalidateAlbumForTrip", () => {
  test("marks the photos query for the given trip as invalidated", () => {
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    seed(qc, "trip-A");
    seed(qc, "trip-B");

    invalidateAlbumForTrip(qc, "trip-A");

    const a = qc.getQueryState(albumKeys.photos("trip-A"));
    const b = qc.getQueryState(albumKeys.photos("trip-B"));
    expect(a?.isInvalidated).toBe(true);
    expect(b?.isInvalidated).toBeFalsy();
  });

  test("is a no-op for unknown trip IDs (does not throw)", () => {
    const qc = new QueryClient();
    expect(() => invalidateAlbumForTrip(qc, "trip-unseen")).not.toThrow();
  });

  test("does not disturb unrelated query namespaces", () => {
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    qc.setQueryData(["finance", "expenses", "trip-A"], []);
    seed(qc, "trip-A");

    invalidateAlbumForTrip(qc, "trip-A");

    const finance = qc.getQueryState(["finance", "expenses", "trip-A"]);
    expect(finance?.isInvalidated).toBeFalsy();
  });
});
