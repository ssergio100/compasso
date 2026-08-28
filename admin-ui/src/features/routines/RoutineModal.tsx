import { AlertTriangle, CalendarDays, Check } from "lucide-react";
import { useState } from "react";
import { Modal, TimeRangePicker } from "../../components";
import type { Routine, RoutineIconKey } from "../../types";
import { inferRoutineIcon, RoutineIconPicker } from "../../visuals";
import { clock, conflictingRoutine, dayNames, routineIconFor, seconds } from "./routineSchedule";

export function RoutineModal({ initial, routines, onClose, onSubmit }: {
  initial?: Routine;
  routines: Routine[];
  onClose: () => void;
  onSubmit: (routine: Omit<Routine, "id">) => Promise<void>;
}) {
  const [name, setName] = useState(initial?.name ?? "");
  const [iconKey, setIconKey] = useState<RoutineIconKey>(initial ? routineIconFor(initial) : "study");
  const [iconTouched, setIconTouched] = useState(Boolean(initial));
  const [start, setStart] = useState(initial ? clock(initial.start_second) : "18:30");
  const [end, setEnd] = useState(initial ? clock(initial.end_second) : "20:00");
  const [days, setDays] = useState(initial ? initial.days.map((selected, index) => selected ? index : -1).filter((index) => index >= 0) : [1, 2, 3, 4, 5]);
  const [busy, setBusy] = useState(false);
  const [serverError, setServerError] = useState("");
  const selectedIcon = iconTouched ? iconKey : inferRoutineIcon(name);
  const draft = {
    name: name.trim(),
    icon_key: selectedIcon,
    start_second: seconds(start),
    end_second: seconds(end),
    days: dayNames.map((_, index) => days.includes(index)) as Routine["days"],
    enabled: initial?.enabled ?? true,
  };
  const conflict = conflictingRoutine(draft, routines);
  const error = conflict ? `Este intervalo já está ocupado pela rotina “${conflict.name}”.` : serverError;

  return <Modal title={initial ? "Editar rotina" : "Nova rotina"} description="Escolha uma identidade, horários e dias." onClose={onClose}><form className="modal-form" onSubmit={async (event) => { event.preventDefault(); if (conflict) return; setBusy(true); setServerError(""); try { await onSubmit(draft); } catch (submitError) { setServerError(submitError instanceof Error ? submitError.message : `Não foi possível ${initial ? "alterar" : "criar"} a rotina.`); } finally { setBusy(false); } }}><label>Nome<input autoFocus placeholder="Ex.: Tempo de estudo" value={name} onChange={(event) => setName(event.target.value)} /></label><RoutineIconPicker value={selectedIcon} onChange={(value) => { setIconKey(value); setIconTouched(true); }} /><TimeRangePicker start={start} end={end} onChange={(nextStart, nextEnd) => { setStart(nextStart); setEnd(nextEnd); setServerError(""); }} /><fieldset className="weekday-picker"><legend>Dias</legend>{dayNames.map((day, index) => { const active = days.includes(index); return <button aria-pressed={active} className={active ? "active" : ""} key={day} onClick={() => { setDays((all) => all.includes(index) ? all.filter((item) => item !== index) : [...all, index]); setServerError(""); }} type="button">{active && <Check size={14} />}{day}</button>; })}</fieldset>{error && <div className="routine-conflict-alert" role="alert"><AlertTriangle aria-hidden="true" size={20} /><div><strong>{conflict ? "Horário indisponível" : `Não foi possível ${initial ? "alterar" : "criar"} a rotina`}</strong><span>{error}</span></div></div>}<div className="modal-actions"><button type="button" onClick={onClose}>Cancelar</button><button className="primary" disabled={!name.trim() || !days.length || Boolean(conflict) || busy}><CalendarDays size={18} />{initial ? "Salvar rotina" : "Criar rotina"}</button></div></form></Modal>;
}
