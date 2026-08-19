import { LockKeyhole, Pause, Play, Plus, UnlockKeyhole } from "lucide-react";
import { useState, type CSSProperties } from "react";
import { Link, useOutletContext } from "react-router-dom";
import { useAppState } from "../../app/AppState";
import { Eyebrow } from "../../components/ui/Eyebrow";
import { OperationStatus } from "../../components/ui/OperationStatus";
import { Surface } from "../../components/ui/Surface";
import type { Client } from "../../domain/models";
import { formatClock, formatMinutes } from "../../lib/formatters";
import { Page } from "../../layouts/Page";
import { BonusTimeDialog } from "./BonusTimeDialog";
import styles from "./TimeRing.module.css";

type RingStyle = CSSProperties & { "--progress": string; "--ring-color": string };

export function UsagePage() {
  const { client } = useOutletContext<{ client: Client }>();
  const {
    addBonusTime,
    getClientControl,
    getClientOperation,
    getRoutines,
    retryClientOperation,
    showToast,
    toggleClientBlocked,
    toggleClientPaused,
  } = useAppState();
  const [bonusTimeOpen, setBonusTimeOpen] = useState(false);
  const routines = getRoutines(client.id);
  const nextRoutine = routines.find((routine) => routine.enabled);
  const control = getClientControl(client.id);
  const operation = getClientOperation(client.id);
  const countingTime = client.agentOnline
    && client.graphicalSessionActive
    && client.monitoringActive
    && client.countingTime
    && !control.paused
    && !control.blocked;
  const effectiveLimit = client.dailyLimitMinutes + control.bonusMinutes;
  const remainingMinutes = client.remainingMinutes;
  const usagePercent = effectiveLimit > 0
    ? Math.min(100, Math.round((client.usedMinutes / effectiveLimit) * 100))
    : 0;
  const remainingPercent = effectiveLimit > 0
    ? Math.max(0, Math.round((remainingMinutes / effectiveLimit) * 100))
    : 0;

  const status = !client.agentOnline
    ? { title: "Agente desconectado", badge: "OFFLINE", badgeClass: "bg-line text-muted", progressClass: "bg-muted", ringColor: "var(--ui-muted)" }
    : client.controlStatus === "block_requested"
      ? { title: "Bloqueio solicitado", badge: "AGUARDANDO", badgeClass: "bg-warning-soft text-warning", progressClass: "bg-warning", ringColor: "var(--ui-warning)" }
    : client.controlStatus === "blocked"
    ? { title: "Uso bloqueado", badge: "BLOQUEADO", badgeClass: "bg-danger-soft text-danger", progressClass: "bg-danger", ringColor: "var(--ui-danger)" }
    : client.controlStatus === "pause_requested"
      ? { title: "Pausa solicitada", badge: "AGUARDANDO", badgeClass: "bg-warning-soft text-warning", progressClass: "bg-warning", ringColor: "var(--ui-warning)" }
    : client.controlStatus === "paused"
      ? { title: "Uso pausado", badge: "PAUSADO", badgeClass: "bg-warning-soft text-warning", progressClass: "bg-warning", ringColor: "var(--ui-warning)" }
      : !client.graphicalSessionActive
          ? { title: "Sem sessão gráfica ativa", badge: "AGENTE ONLINE", badgeClass: "bg-brand-soft text-brand", progressClass: "bg-muted", ringColor: "var(--ui-muted)" }
          : client.monitoringActive
        ? { title: "Uso liberado", badge: "ONLINE", badgeClass: "bg-brand-soft text-brand", progressClass: "bg-brand", ringColor: "var(--ui-brand)" }
        : { title: "Monitoramento indisponível", badge: "ATENÇÃO", badgeClass: "bg-warning-soft text-warning", progressClass: "bg-warning", ringColor: "var(--ui-warning)" };

  async function handlePause() {
    const changed = await toggleClientPaused(client.id);
    showToast(changed ? (control.paused ? "Uso retomado." : "Uso pausado.") : "Não foi possível enviar o comando.");
  }

  async function handleBlock() {
    const changed = await toggleClientBlocked(client.id);
    showToast(changed ? (control.blocked ? "Bloqueio removido." : "Bloqueio ativado.") : "Não foi possível enviar o comando.");
  }

  return (
    <Page>
      <div className="grid items-start gap-5 md:grid-cols-[minmax(0,1.35fr)_minmax(16.25rem,.65fr)]">
        <Surface className="overflow-hidden p-5 md:p-6">
          <div className="flex items-start justify-between gap-4">
            <div><Eyebrow>O que está acontecendo</Eyebrow><h1 className="text-xl font-extrabold">{status.title}</h1></div>
            <span className={`rounded-full px-2.5 py-1.5 text-xs font-extrabold ${status.badgeClass}`}>{status.badge}</span>
          </div>
          <div className="my-6 grid grid-cols-[7.25rem_minmax(0,1fr)] items-center gap-4">
            <div className={styles.ring} style={{ "--progress": `${remainingPercent}%`, "--ring-color": status.ringColor } as RingStyle}>
              <div className={styles.content}><strong className="numbers-tabular block text-xl tracking-[-.04em]">{formatClock(remainingMinutes)}</strong><span className="mt-0.5 block text-xs text-muted">restantes</span></div>
            </div>
            <div>
              <strong className="block text-base leading-tight">{remainingPercent}% do tempo disponível</strong>
              <span className="mt-2 block text-sm leading-relaxed text-muted">{formatMinutes(client.usedMinutes)} utilizados de {formatMinutes(effectiveLimit)} disponíveis hoje.</span>
              <div aria-label={`${usagePercent}% do limite utilizado`} className="mt-4 h-2 overflow-hidden rounded-full bg-line" role="progressbar" aria-valuemax={100} aria-valuemin={0} aria-valuenow={usagePercent}><span className={`block h-full rounded-full ${status.progressClass}`} style={{ width: `${usagePercent}%` }} /></div>
            </div>
          </div>
          <div className="grid grid-cols-3 gap-2.5">
            <button className="grid min-h-[5rem] place-items-center content-center gap-2 rounded-2xl border border-line p-2 text-xs font-extrabold text-ink transition-[transform,border-color] hover:border-brand/45 active:scale-[.97]" onClick={() => setBonusTimeOpen(true)} type="button"><span className="grid size-8 place-items-center rounded-[.65rem] bg-brand-soft text-brand"><Plus aria-hidden="true" size={18} /></span>Mais tempo</button>
            <button aria-pressed={control.paused} className={`grid min-h-[5rem] place-items-center content-center gap-2 rounded-2xl border p-2 text-xs font-extrabold transition-[transform,background-color,border-color,color] active:scale-[.97] ${control.paused ? "border-warning bg-warning-soft text-warning" : "border-line text-ink hover:border-warning/45"}`} onClick={handlePause} type="button"><span className={`grid size-8 place-items-center rounded-[.65rem] ${control.paused ? "bg-warning text-white" : "bg-brand-soft text-brand"}`}>{control.paused ? <Play aria-hidden="true" size={18} /> : <Pause aria-hidden="true" size={18} />}</span>{control.paused ? "Retomar" : "Pausar"}</button>
            <button aria-pressed={control.blocked} className={`grid min-h-[5rem] place-items-center content-center gap-2 rounded-2xl border p-2 text-xs font-extrabold transition-[transform,background-color,border-color,color] active:scale-[.97] ${control.blocked ? "border-danger bg-danger text-white" : "border-danger/20 text-danger hover:border-danger/45"}`} onClick={handleBlock} type="button"><span className={`grid size-8 place-items-center rounded-[.65rem] ${control.blocked ? "bg-white/20 text-white" : "bg-danger-soft text-danger"}`}>{control.blocked ? <UnlockKeyhole aria-hidden="true" size={18} /> : <LockKeyhole aria-hidden="true" size={18} />}</span>{control.blocked ? "Desbloquear" : "Bloquear"}</button>
          </div>
          <OperationStatus operation={operation} onRetry={() => retryClientOperation(client.id)} />
        </Surface>
        <div className="md:pt-5">
          <div className="mb-3 flex min-h-11 items-end justify-between gap-4 px-0.5 md:mt-0"><h2 className="text-lg font-extrabold">Resumo de hoje</h2><Link className="grid min-h-11 place-items-center text-sm font-bold text-brand hover:underline" to={`/clients/${client.id}/limits`}>Editar</Link></div>
          <Surface className="px-4 py-1"><dl>
            <div className="flex items-center justify-between gap-4 border-b border-line py-4"><dt className="text-sm text-muted">Limite diário</dt><dd className="text-right text-sm font-bold">{formatMinutes(client.dailyLimitMinutes)}</dd></div>
            {control.bonusMinutes > 0 && <div className="flex items-center justify-between gap-4 border-b border-line py-4"><dt className="text-sm text-muted">Tempo adicional</dt><dd className="text-right text-sm font-bold text-brand">+{formatMinutes(control.bonusMinutes)}</dd></div>}
            <div className="flex items-center justify-between gap-4 border-b border-line py-4"><dt className="text-sm text-muted">Próxima rotina</dt><dd className="text-right text-sm font-bold">{nextRoutine ? `${nextRoutine.name} · ${nextRoutine.start}` : "Nenhuma ativa"}</dd></div>
            <div className="flex items-center justify-between gap-4 py-4"><dt className="text-sm text-muted">Última sincronização</dt><dd className="text-right text-sm font-bold">{client.lastSynchronization}</dd></div>
          </dl></Surface>
          <h2 className="mt-5 mb-3 px-0.5 text-lg font-extrabold">Estado do computador</h2>
          <Surface className="px-4 py-1"><dl>
            <StatusRow active={client.agentOnline} activeText="Conectado" inactiveText="Desconectado" label="Agente" />
            <StatusRow active={client.graphicalSessionActive} activeText="Ativa" inactiveText="Sem sessão" label="Sessão gráfica" />
            <StatusRow active={client.monitoringActive && !control.paused} activeText="Ativo" inactiveText={control.paused ? "Pausado" : "Inativo"} label="Monitoramento" />
            <StatusRow active={countingTime} activeText="Em andamento" inactiveText="Parada" label="Contagem de tempo" last />
          </dl></Surface>
        </div>
      </div>
      <BonusTimeDialog clientName={client.name} onClose={() => setBonusTimeOpen(false)} onConfirm={async (minutes) => { const changed = await addBonusTime(client.id, minutes); showToast(changed ? "Tempo registrado. Sincronizando com o computador…" : "Não foi possível adicionar o tempo."); }} open={bonusTimeOpen} />
    </Page>
  );
}

function StatusRow({ active, activeText, inactiveText, label, last = false }: { active: boolean; activeText: string; inactiveText: string; label: string; last?: boolean }) {
  return <div className={`flex items-center justify-between gap-4 py-3.5 ${last ? "" : "border-b border-line"}`}><dt className="text-sm text-muted">{label}</dt><dd className="inline-flex items-center gap-2 text-right text-sm font-bold"><span aria-hidden="true" className={`size-2 rounded-full ${active ? "bg-emerald-500" : "bg-neutral-400"}`} />{active ? activeText : inactiveText}</dd></div>;
}
