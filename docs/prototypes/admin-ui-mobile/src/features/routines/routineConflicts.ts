import type { Routine, RoutineDraft } from "../../domain/models";

const minutesPerDay = 24 * 60;
const minutesPerWeek = 7 * minutesPerDay;

interface Interval {
  start: number;
  end: number;
}

function parseClock(value: string) {
  const [hours, minutes] = value.split(":").map(Number);
  return hours * 60 + minutes;
}

function weeklyIntervals(routine: Pick<RoutineDraft, "days" | "start" | "end">): Interval[] {
  const startMinute = parseClock(routine.start);
  const endMinute = parseClock(routine.end);
  const fullDay = startMinute === endMinute;
  const intervals: Interval[] = [];

  for (const day of routine.days) {
    for (const weekOffset of [-minutesPerWeek, 0, minutesPerWeek]) {
      const dayStart = day * minutesPerDay + weekOffset;
      const start = fullDay ? dayStart : dayStart + startMinute;
      const end = fullDay
        ? dayStart + minutesPerDay
        : endMinute > startMinute
          ? dayStart + endMinute
          : dayStart + minutesPerDay + endMinute;
      intervals.push({ start, end });
    }
  }

  return intervals;
}

function overlaps(first: Interval, second: Interval) {
  return Math.max(first.start, second.start) < Math.min(first.end, second.end);
}

export function findRoutineConflict(
  draft: Pick<RoutineDraft, "days" | "start" | "end">,
  routines: Routine[],
  excludedRoutineId?: string,
) {
  const candidateIntervals = weeklyIntervals(draft);

  return routines.find((routine) => {
    if (!routine.enabled || routine.id === excludedRoutineId) return false;
    const existingIntervals = weeklyIntervals(routine);
    return candidateIntervals.some((candidate) => (
      existingIntervals.some((existing) => overlaps(candidate, existing))
    ));
  });
}
