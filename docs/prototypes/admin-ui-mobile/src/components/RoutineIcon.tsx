import {
  BookOpen,
  Dumbbell,
  Focus,
  Footprints,
  Gamepad2,
  GraduationCap,
  House,
  Moon,
  Music2,
  Sparkles,
  Tv,
  Utensils,
  type LucideIcon,
} from "lucide-react";
import type { RoutineIconName } from "../domain/models";

export const routineIconOptions: Array<{ name: RoutineIconName; label: string; Icon: LucideIcon }> = [
  { name: "moon", label: "Dormir", Icon: Moon },
  { name: "book-open", label: "Estudar", Icon: BookOpen },
  { name: "graduation-cap", label: "Aula", Icon: GraduationCap },
  { name: "utensils", label: "Refeição", Icon: Utensils },
  { name: "gamepad", label: "Jogar", Icon: Gamepad2 },
  { name: "tv", label: "Assistir", Icon: Tv },
  { name: "dumbbell", label: "Exercício", Icon: Dumbbell },
  { name: "music", label: "Música", Icon: Music2 },
  { name: "house", label: "Em casa", Icon: House },
  { name: "footprints", label: "Sair", Icon: Footprints },
  { name: "sparkles", label: "Livre", Icon: Sparkles },
  { name: "focus", label: "Foco", Icon: Focus },
];

export function RoutineIcon({ name, muted = false, size = 46 }: { name: RoutineIconName; muted?: boolean; size?: number }) {
  const option = routineIconOptions.find((item) => item.name === name) ?? routineIconOptions[0];
  return (
    <span
      aria-hidden="true"
      className={`grid shrink-0 place-items-center rounded-[.9rem] ${muted ? "bg-line text-muted" : "bg-brand-soft text-brand"}`}
      style={{ width: size, height: size }}
    >
      <option.Icon size={Math.round(size * 0.46)} strokeWidth={2.2} />
    </span>
  );
}
