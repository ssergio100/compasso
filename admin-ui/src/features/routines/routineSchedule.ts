import type { Routine, RoutineIconKey } from "../../types";
import { inferRoutineIcon } from "../../visuals";

export const dayNames = ["Dom", "Seg", "Ter", "Qua", "Qui", "Sex", "Sáb"];

export function clock(seconds: number) {
  const total = Math.floor(seconds / 60);
  return `${String(Math.floor(total / 60) % 24).padStart(2, "0")}:${String(total % 60).padStart(2, "0")}`;
}

export function seconds(value: string) {
  const [hours, minutes] = value.split(":").map(Number);
  return hours * 3600 + minutes * 60;
}

export function withoutId(routine: Routine): Omit<Routine, "id"> {
  const { id: _, ...value } = routine;
  return value;
}

export function routineIconFor(routine: Routine): RoutineIconKey {
  return routine.icon_key ?? inferRoutineIcon(routine.name);
}

export function routineIntervals(routine: Omit<Routine, "id">): [number, number][] {
  const daySeconds = 86400;
  const intervals: [number, number][] = [];
  for (let day = 0; day < 7; day += 1) {
    const previous = (day + 6) % 7;
    if (routine.start_second === routine.end_second) {
      if (routine.days[day]) intervals.push([day * daySeconds, (day + 1) * daySeconds]);
    } else if (routine.start_second < routine.end_second) {
      if (routine.days[day]) intervals.push([day * daySeconds + routine.start_second, day * daySeconds + routine.end_second]);
    } else {
      if (routine.days[previous] && routine.end_second > 0) intervals.push([day * daySeconds, day * daySeconds + routine.end_second]);
      if (routine.days[day]) intervals.push([day * daySeconds + routine.start_second, (day + 1) * daySeconds]);
    }
  }
  return intervals;
}

export function conflictingRoutine(draft: Omit<Routine, "id">, routines: Routine[]): Routine | undefined {
  const intervals = routineIntervals(draft);
  return routines.find((routine) => routineIntervals(routine).some(([start, end]) => intervals.some(([draftStart, draftEnd]) => start < draftEnd && draftStart < end)));
}

export interface RoutineSegment {
  key: string;
  name: string;
  start: number;
  end: number;
}

export function nextRoutineLabel(routines: Routine[], now: Date) {
  let next: { routine: Routine; at: Date; offset: number } | undefined;
  for (let offset = 0; offset <= 7; offset += 1) {
    for (const routine of routines) {
      if (!routine.enabled) continue;
      const at = new Date(now);
      at.setDate(now.getDate() + offset);
      if (!routine.days[at.getDay()]) continue;
      const start = routine.start_second === routine.end_second ? 0 : routine.start_second;
      at.setHours(Math.floor(start / 3600), Math.floor(start / 60) % 60, start % 60, 0);
      if (at <= now || next && at >= next.at) continue;
      next = { routine, at, offset };
    }
  }
  if (!next) return "Nenhuma programada";
  const day = next.offset === 0 ? "Hoje" : next.offset === 1 ? "Amanhã" : dayNames[next.at.getDay()];
  return `${next.routine.name} · ${day}, ${clock(next.routine.start_second === next.routine.end_second ? 0 : next.routine.start_second)}`;
}

export function routineSegmentsForDay(routines: Routine[], day: number): RoutineSegment[] {
  const previousDay = (day + 6) % 7;
  return routines.flatMap((routine) => {
    if (!routine.enabled) return [];
    const segment = (start: number, end: number, edge: string): RoutineSegment => ({
      key: `${routine.id}-${edge}`,
      name: routine.name,
      start,
      end,
    });
    if (routine.start_second === routine.end_second) {
      return routine.days[day] ? [segment(0, 86400, "full")] : [];
    }
    if (routine.start_second < routine.end_second) {
      return routine.days[day] ? [segment(routine.start_second, routine.end_second, "day")] : [];
    }
    const overnight: RoutineSegment[] = [];
    if (routine.days[previousDay] && routine.end_second > 0) overnight.push(segment(0, routine.end_second, "morning"));
    if (routine.days[day]) overnight.push(segment(routine.start_second, 86400, "night"));
    return overnight;
  });
}
