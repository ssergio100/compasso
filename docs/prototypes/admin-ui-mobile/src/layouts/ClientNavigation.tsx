import { CalendarDays, Clock3, House, Settings } from "lucide-react";
import { NavLink, useParams } from "react-router-dom";

const items = [
  { section: "now", label: "Agora", Icon: House },
  { section: "limits", label: "Limites", Icon: Clock3 },
  { section: "routines", label: "Rotinas", Icon: CalendarDays },
  { section: "administration", label: "Administração", shortLabel: "Admin.", Icon: Settings },
];

export function ClientNavigation() {
  const { clientId } = useParams();

  return (
    <footer className="safe-bottom fixed right-0 bottom-0 left-0 z-20 border-t border-line bg-surface/96 px-2 pt-1.5 backdrop-blur-md md:bottom-4 md:left-1/2 md:w-[min(38.75rem,calc(100%-2rem))] md:-translate-x-1/2 md:rounded-[1.125rem] md:border md:shadow-panel">
      <nav aria-label="Seções do cliente" className="mx-auto grid max-w-[61.25rem] grid-cols-4">
        {items.map(({ section, label, shortLabel, Icon }) => (
          <NavLink
            aria-label={label}
            className={({ isActive }) => `grid min-h-14 min-w-0 place-items-center content-center gap-1 rounded-xl px-1 text-[.7rem] transition-colors ${isActive ? "font-extrabold text-brand" : "text-muted hover:text-ink"}`}
            key={section}
            to={`/clients/${clientId}/${section}`}
          >
            <Icon aria-hidden="true" size={20} strokeWidth={2.1} />
            <span>{shortLabel ?? label}</span>
          </NavLink>
        ))}
      </nav>
    </footer>
  );
}
