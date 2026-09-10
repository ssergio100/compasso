import { useRef, useState } from "react";
import { DurationWheel } from "../../components";
import type { Device } from "../../types";
import { formatDuration } from "../common/format";
import { dayNames } from "../routines/routineSchedule";

const fullDayNames = [
  "Domingo",
  "Segunda-feira",
  "Terça-feira",
  "Quarta-feira",
  "Quinta-feira",
  "Sexta-feira",
  "Sábado",
];

export function LimitsPage({ device, onSave }: { device: Device; onSave: (weekly: number[]) => Promise<void> }) {
  const [draft, setDraft] = useState([...device.weekly_quota_seconds]);
  const draftRef = useRef(draft);
  const [selectedDay, setSelectedDay] = useState(new Date().getDay());
  const [busy, setBusy] = useState(false);
  const value = draft[selectedDay];
  const updateSelectedDay = (nextValue: number) => {
    const nextDraft = draftRef.current.map((item, index) => index === selectedDay ? nextValue : item);
    draftRef.current = nextDraft;
    setDraft(nextDraft);
  };
  const saveSelectedDay = async (nextValue: number) => {
    const nextDraft = draftRef.current.map((item, index) => index === selectedDay ? nextValue : item);
    draftRef.current = nextDraft;
    setDraft(nextDraft);
    setBusy(true);
    try { await onSave(nextDraft); } finally { setBusy(false); }
  };

  return <section className="editor-page"><header><div><h2>Limites</h2><p>Selecione um dia para definir o tempo disponível.</p></div></header><div className="day-tabs">{dayNames.map((name, index) => { const selected = selectedDay === index; return <button aria-pressed={selected} className={selected ? "active" : ""} key={name} onClick={() => setSelectedDay(index)}><span>{name}</span><strong>{formatDuration(draft[index])}</strong></button>; })}</div><section aria-busy={busy} className="limit-editor"><DurationWheel caption={fullDayNames[selectedDay]} label="Limite" value={value} onChange={updateSelectedDay} onCommit={saveSelectedDay} /><div className="step-actions"><button disabled={busy} type="button" onClick={() => saveSelectedDay(0)}>Bloquear o dia</button><button disabled={busy} type="button" onClick={() => saveSelectedDay(86400)}>Liberar o dia</button></div></section></section>;
}
