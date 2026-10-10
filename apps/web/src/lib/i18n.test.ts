import { beforeEach, describe, expect, it } from "vitest";
import { getLang, t, tStatus } from "./i18n";

describe("i18n", () => {
  beforeEach(() => localStorage.clear());

  it("translates field-app keys in every language", () => {
    expect(t("jobs.checkIn", "en")).toBe("Check in");
    expect(t("jobs.checkIn", "th")).toBe("ลงเวลาเข้างาน");
    expect(t("jobs.checkIn", "my")).toBe("အလုပ်ဝင်ချိန်မှတ်မည်");
  });

  it("translates booking statuses and falls back for unknown ones", () => {
    expect(tStatus("in_progress", "th")).toBe("กำลังทำ");
    expect(tStatus("mystery_state", "th")).toBe("mystery state");
  });

  it("remembers the chosen language", () => {
    localStorage.setItem("smileclean.lang", "my");
    expect(getLang()).toBe("my");
    localStorage.setItem("smileclean.lang", "xx");
    expect(["en", "th", "my"]).toContain(getLang());
  });
});
