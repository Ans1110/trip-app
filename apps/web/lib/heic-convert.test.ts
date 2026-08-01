import { describe, expect, test } from "bun:test";

import { isHeic, renameToJpeg } from "./heic-convert";

describe("isHeic", () => {
  test("detects by image/heic mime", () => {
    expect(isHeic({ name: "photo", type: "image/heic" })).toBe(true);
  });

  test("detects by image/heif mime", () => {
    expect(isHeic({ name: "photo", type: "image/heif" })).toBe(true);
  });

  test("detects by .heic extension when mime is blank", () => {
    // Some bridges (Windows Photos, Whatsapp) strip the type but keep the name.
    expect(isHeic({ name: "IMG_1234.HEIC", type: "" })).toBe(true);
  });

  test("detects by .heif extension case-insensitively", () => {
    expect(isHeic({ name: "photo.HeiF", type: "" })).toBe(true);
  });

  test("rejects jpeg by mime", () => {
    expect(isHeic({ name: "photo.heic.jpg", type: "image/jpeg" })).toBe(false);
  });

  test("rejects png without heic extension", () => {
    expect(isHeic({ name: "photo.png", type: "image/png" })).toBe(false);
  });

  test("does not confuse .heic in the middle of a name", () => {
    // Extension check is anchored to the last dot.
    expect(isHeic({ name: "heic-photo.png", type: "image/png" })).toBe(false);
  });
});

describe("renameToJpeg", () => {
  test("swaps the extension", () => {
    expect(renameToJpeg("IMG_1234.HEIC")).toBe("IMG_1234.jpg");
  });

  test("keeps stem when it contains dots", () => {
    expect(renameToJpeg("my.photo.heif")).toBe("my.photo.jpg");
  });

  test("appends .jpg when there is no extension", () => {
    expect(renameToJpeg("nameonly")).toBe("nameonly.jpg");
  });

  test("appends .jpg when the leading char is a dot", () => {
    // Files like `.hidden` are treated as extensionless — we don't want to
    // strip everything and produce `.jpg`.
    expect(renameToJpeg(".hidden")).toBe(".hidden.jpg");
  });
});
