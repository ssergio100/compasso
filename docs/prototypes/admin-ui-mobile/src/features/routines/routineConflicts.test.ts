import type { Routine, RoutineDraft } from "../../domain/models";
import { findRoutineConflict } from "./routineConflicts";

function routine(overrides: Partial<Routine> = {}): Routine {
  return {
    id: "existing",
    clientId: "quarto",
    name: "Existente",
    icon: "moon",
    start: "22:00",
    end: "07:00",
    days: [6],
    enabled: true,
    ...overrides,
  };
}

function draft(overrides: Partial<RoutineDraft> = {}): RoutineDraft {
  return {
    name: "Nova",
    icon: "book-open",
    start: "06:00",
    end: "08:00",
    days: [0],
    ...overrides,
  };
}

describe("routine conflict detection", () => {
  it("detects a conflict after midnight, including the weekly boundary", () => {
    expect(findRoutineConflict(draft(), [routine()])?.name).toBe("Existente");
  });

  it("allows routines whose end and start touch at the same instant", () => {
    const existing = routine({ start: "18:00", end: "20:00", days: [1] });
    const candidate = draft({ start: "20:00", end: "22:00", days: [1] });

    expect(findRoutineConflict(candidate, [existing])).toBeUndefined();
  });

  it("treats equal start and end as a full-day routine", () => {
    const existing = routine({ start: "10:00", end: "10:00", days: [2] });
    const candidate = draft({ start: "14:00", end: "15:00", days: [2] });

    expect(findRoutineConflict(candidate, [existing])?.id).toBe(existing.id);
  });

  it("ignores inactive routines and the routine being edited", () => {
    const existing = routine({ enabled: false });
    expect(findRoutineConflict(draft(), [existing])).toBeUndefined();

    existing.enabled = true;
    expect(findRoutineConflict(draft(), [existing], existing.id)).toBeUndefined();
  });
});
