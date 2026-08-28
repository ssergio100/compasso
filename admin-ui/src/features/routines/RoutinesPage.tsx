import { CalendarDays, Pencil, Plus, Trash2 } from "lucide-react";
import type { Device, Routine } from "../../types";
import { RoutineVisual } from "../../visuals";
import { clock, dayNames, routineIconFor } from "./routineSchedule";

export function RoutinesPage({ device, onNew, onEdit, onToggle, onDelete }: {
  device: Device;
  onNew: () => void;
  onEdit: (routine: Routine) => void;
  onToggle: (routine: Routine) => Promise<void>;
  onDelete: (routine: Routine) => Promise<void>;
}) {
  return <section className="editor-page"><header><div><h2>Rotinas</h2><p>Crie ritmos automáticos para estudo, descanso e lazer.</p></div><button className="primary-button" onClick={onNew}><Plus size={18} />Nova rotina</button></header><div className="routine-list">{device.routines.length ? device.routines.map((routine) => <article key={routine.id}><RoutineVisual iconKey={routineIconFor(routine)} /><span className="routine-time">{clock(routine.start_second)}<i />{clock(routine.end_second)}</span><div><h3>{routine.name}</h3><p>{routine.days.map((selected, index) => selected ? dayNames[index] : "").filter(Boolean).join(" · ")}</p></div><button aria-pressed={routine.enabled} className={`switch ${routine.enabled ? "active" : ""}`} onClick={() => void onToggle(routine)}><i /></button><button aria-label={`Editar ${routine.name}`} className="edit-routine" onClick={() => onEdit(routine)}><Pencil size={18} /></button><button aria-label={`Excluir ${routine.name}`} className="delete" onClick={() => void onDelete(routine)}><Trash2 size={18} /></button></article>) : <div className="empty"><CalendarDays size={28} /><h3>Nenhuma rotina criada</h3><p>Comece organizando um período recorrente.</p></div>}</div></section>;
}
