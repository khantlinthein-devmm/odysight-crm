import { describe, expect, it } from "vitest";
import { compressPhoto, scaledSize } from "./image";
import { checklistState } from "./checklists";

describe("scaledSize", () => {
  it("fits the longest edge into 1920 px and keeps the aspect ratio", () => {
    expect(scaledSize(4032, 3024)).toEqual({ width: 1920, height: 1440 });
    expect(scaledSize(3024, 4032)).toEqual({ width: 1440, height: 1920 });
  });

  it("leaves small photos alone", () => {
    expect(scaledSize(1280, 960)).toEqual({ width: 1280, height: 960 });
  });
});

describe("compressPhoto", () => {
  it("returns small or non-image files unchanged", async () => {
    const small = new File(["x"], "a.jpg", { type: "image/jpeg" });
    expect(await compressPhoto(small)).toBe(small);
    const pdf = new File([new Uint8Array(2_000_000)], "a.pdf", { type: "application/pdf" });
    expect(await compressPhoto(pdf)).toBe(pdf);
  });
});

describe("checklistState", () => {
  it("flags a completed job without a submitted checklist as missing", () => {
    expect(checklistState({ status: "pending", reviewStatus: "", overdue: true })).toBe("missing");
    expect(checklistState({ status: "completed", reviewStatus: "", submittedAt: "2026-10-15" })).toBe("submitted");
  });
});
