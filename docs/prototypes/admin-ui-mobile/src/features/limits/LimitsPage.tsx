import { Ban, Check, Save } from "lucide-react";
import { useEffect, useState, type CSSProperties } from "react";
import { useOutletContext } from "react-router-dom";
import { useAppState } from "../../app/AppState";
import { Button } from "../../components/ui/Button";
import { Eyebrow } from "../../components/ui/Eyebrow";
import { OperationStatus } from "../../components/ui/OperationStatus";
import { Surface } from "../../components/ui/Surface";
import type { Client } from "../../domain/models";
import { formatClock, formatMinutes } from "../../lib/formatters";
import { Page } from "../../layouts/Page";
import styles from "./LimitDial.module.css";

const days = ["Dom", "Seg", "Ter", "Qua", "Qui", "Sex", "Sáb"];

export function LimitsPage() {
  const { client } = useOutletContext<{ client: Client }>();
  const { getClientOperation, limits, retryClientOperation, saveLimits, showToast } = useAppState();
  const [selectedDay, setSelectedDay] = useState(1);
  const [saving, setSaving] = useState(false);
  const savedLimits = limits[client.id] ?? [0, 0, 0, 0, 0, 0, 0];
  const [draftLimits, setDraftLimits] = useState(() => [...savedLimits]);
  const value = draftLimits[selectedDay];
  const changed = draftLimits.some((limit, index) => limit !== savedLimits[index]);
  const operation = getClientOperation(client.id);

  useEffect(() => setDraftLimits([...savedLimits]), [client.id, limits]);

  const setLimit = (minutes: number) => setDraftLimits((current) => (
    current.map((limit, index) => index === selectedDay ? minutes : limit)
  ));

  const save = async () => {
    if (!changed) return;
    setSaving(true);
    const saved = await saveLimits(client.id, draftLimits);
    setSaving(false);
    showToast(saved
      ? (client.agentOnline ? "Limites salvos e enviados ao agente." : "Limites salvos. Aguardando o agente conectar.")
      : "Não foi possível salvar os limites.");
  };

  return (
    <Page narrow>
      <Surface className="p-5 md:p-6">
        <Eyebrow>Regra semanal</Eyebrow>
        <h1 className="text-xl font-extrabold">Limites de tempo diários</h1>
        <p className="mt-2 text-sm leading-relaxed text-muted">Selecione um dia e defina quanto tempo poderá ser usado.</p>

        <div aria-label="Dias da semana" className="mt-5 grid grid-cols-4 gap-2 sm:grid-cols-7 sm:gap-1.5" role="tablist">
          {days.map((day, index) => {
            const active = selectedDay === index;
            const blocked = draftLimits[index] === 0;
            return (
              <button
                aria-selected={active}
                className={`grid min-h-[3.5rem] min-w-11 place-items-center content-center rounded-xl border px-0.5 py-2 transition-colors ${active ? blocked ? "border-danger bg-danger text-white" : "border-brand bg-brand text-white" : blocked ? "border-danger/25 bg-danger-soft/50 text-danger" : "border-line bg-surface text-muted hover:border-brand/40"}`}
                key={day}
                onClick={() => setSelectedDay(index)}
                role="tab"
                type="button"
              >
                <strong className="text-[.7rem] sm:text-xs">{day}</strong>
                <span className="numbers-tabular mt-1 text-[.65rem] sm:text-xs">{formatMinutes(draftLimits[index]).replace(" ", "")}</span>
              </button>
            );
          })}
        </div>

        <div className="my-7 flex justify-center">
          <div className={styles.dial} style={{ "--progress": `${value / 14.4}%` } as CSSProperties}>
            <div className={styles.content}>
              <span className="block text-xs font-bold text-muted">Tempo disponível</span>
              <strong className="numbers-tabular mt-1 block text-3xl font-extrabold tracking-[-.04em]">{formatClock(value)}</strong>
            </div>
          </div>
        </div>

        <input
          aria-label={`Tempo disponível em ${days[selectedDay]}`}
          className={styles.slider}
          max="1440"
          min="0"
          onChange={(event) => setLimit(Number(event.currentTarget.value))}
          step="15"
          type="range"
          value={value}
        />

        <div className="mt-4 grid grid-cols-2 gap-2.5">
          <Button leadingIcon={<Check aria-hidden="true" size={17} />} onClick={() => setLimit(1440)} variant="secondary">Liberar o dia</Button>
          <Button leadingIcon={<Ban aria-hidden="true" size={17} />} onClick={() => setLimit(0)} variant="danger">Bloquear o dia</Button>
        </div>
        {changed ? <p className="mt-4 text-center text-xs font-bold text-warning" role="status">Alterações ainda não salvas</p> : null}
        <div className="mt-4 grid grid-cols-2 gap-2.5">
          <Button disabled={!changed || saving} onClick={() => setDraftLimits([...savedLimits])} variant="secondary">Cancelar</Button>
          <Button disabled={!changed || saving} leadingIcon={<Save aria-hidden="true" size={17} />} onClick={save}>{saving ? "Salvando…" : "Salvar limites"}</Button>
        </div>
        <OperationStatus operation={operation?.label === "Atualizar limites" ? operation : undefined} onRetry={() => retryClientOperation(client.id)} />
      </Surface>
    </Page>
  );
}
