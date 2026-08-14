const shortDays = ["Dom", "Seg", "Ter", "Qua", "Qui", "Sex", "Sáb"];

export function formatMinutes(minutes: number) {
  if (minutes === 0) return "0h";
  if (minutes === 1440) return "24h";
  const hours = Math.floor(minutes / 60);
  const remainder = minutes % 60;
  return remainder ? `${hours}h ${String(remainder).padStart(2, "0")}min` : `${hours}h`;
}

export function formatClock(minutes: number) {
  const hours = Math.floor(minutes / 60);
  const remainder = minutes % 60;
  return `${String(hours).padStart(2, "0")}:${String(remainder).padStart(2, "0")}`;
}

export function formatRoutineDays(days: number[]) {
  if (days.length === 7) return "Todos os dias";
  const weekdays = [1, 2, 3, 4, 5];
  if (weekdays.every((day) => days.includes(day)) && days.length === weekdays.length) return "Seg–Sex";
  if ([0, 1, 2, 3, 4].every((day) => days.includes(day)) && days.length === 5) return "Dom–Qui";
  return days.map((day) => shortDays[day]).join(", ");
}
