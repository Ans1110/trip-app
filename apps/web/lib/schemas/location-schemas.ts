import { z } from "zod";

const lat = z.number().min(-90).max(90);
const lng = z.number().min(-180).max(180);

const optStr = (max: number, label = "Too long") =>
  z.string().trim().max(max, label);

export const savedPlaceSchema = z.object({
  name: z.string().trim().min(1, "Required").max(120, "Too long"),
  address: optStr(240),
  lat,
  lng,
  place_id: optStr(120),
  category: optStr(32),
  notes: optStr(1000),
  color: optStr(16),
  trip_id: optStr(64),
});

export type SavedPlaceFormInput = z.infer<typeof savedPlaceSchema>;

export const searchSchema = z.object({
  q: z.string().trim().min(1, "Required").max(120, "Too long"),
});
export type SearchFormInput = z.infer<typeof searchSchema>;

export const routeSchema = z
  .object({
    origin_lat: lat,
    origin_lng: lng,
    destination_lat: lat,
    destination_lng: lng,
    mode: z.enum(["driving", "walking", "bicycling", "transit"]),
  })
  .refine(
    (v) =>
      !(
        v.origin_lat === v.destination_lat &&
        v.origin_lng === v.destination_lng
      ),
    { message: "Origin and destination are identical", path: ["destination_lat"] },
  );

export type RouteFormInput = z.infer<typeof routeSchema>;
