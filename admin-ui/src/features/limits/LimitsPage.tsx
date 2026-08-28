import { Save } from "lucide-react";
import { useState } from "react";
import { DurationWheel } from "../../components";
import type { Device } from "../../types";
import { formatDuration } from "../common/format";
import { dayNames } from "../routines/routineSchedule";

export function LimitsPage({ device, onSave }: { device: Device; onSave: (weekly: number[]) => Promise<void> }) {
  const [draft, setDraft] = useState([...device.weekly_quota_seconds]);
  const [selectedDays, setSelectedDays] = useState([new Date().getDay()]);
  const [busy, setBusy] = useState(false);
  const value = draft[selectedDays[0]];
  const multipleDays = selectedDays.length > 1;
  const updateSelectedDays = (nextValue: number) => {
    setDraft((all) => all.map((item, index) => selectedDays.includes(index) ? nextValue : item));
  };
  const toggleDay = (day: number) => {
    setSelectedDays((all) => {
      if (!all.includes(day)) return [...all, day].sort((left, right) => left - right);
      if (all.length === 1) return all;
      return all.filter((item) => item !== day);
    });
  };

  return <section className="editor-page"><header><div><h2>Limites</h2><p>Selecione um ou mais dias para definir o mesmo tempo disponível.</p></div></header><div className="day-tabs">{dayNames.map((name, index) => { const selected = selectedDays.includes(index); return <button aria-pressed={selected} className={selected ? "active" : ""} key={name} onClick={() => toggleDay(index)}><span>{name}</span><strong>{formatDuration(draft[index])}</strong></button>; })}</div><section className="limit-editor"><DurationWheel label={multipleDays ? `${selectedDays.length} dias selecionados` : `Limite · ${dayNames[selectedDays[0]]}`} value={value} onChange={updateSelectedDays} /><div className="step-actions"><button type="button" onClick={() => updateSelectedDays(0)}>{multipleDays ? "Bloquear os dias" : "Bloquear o dia"}</button><button type="button" onClick={() => updateSelectedDays(86400)}>{multipleDays ? "Liberar os dias" : "Liberar o dia"}</button></div><button className="primary-button" disabled={busy} onClick={async () => { setBusy(true); try { await onSave(draft); } finally { setBusy(false); } }}><Save size={18} />{busy ? "Salvando…" : "Salvar limites"}</button></section></section>;
}
