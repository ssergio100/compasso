import { AlertTriangle, ChevronDown, ChevronRight, Radio, Search, Trash2 } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { api, remoteMode } from "../api";
import { Modal } from "../components";
import type { CommunicationLog, CommunicationParty, CommunicationResult } from "../types";
import { mockCommunication } from "./mockCommunication";

type LiveState = "connecting" | "live" | "interrupted";
type PartyFilter = "all" | CommunicationParty;
type ResultFilter = "all" | CommunicationResult;

const partyNames: Record<CommunicationParty, string> = { agent: "Agente", api: "API", interface: "Interface" };
const operationNames: Record<string, string> = {
  heartbeat: "Heartbeat",
  heartbeat_response: "Resposta do heartbeat",
  policy_response: "Resposta de políticas",
  bonus_acknowledged: "Bônus confirmado",
  session_state: "Estado da sessão",
  "GET device": "Atualização do computador",
  "GET status": "Consulta de estado",
  "GET commands": "Consulta da operação",
  "POST bonus": "Bônus enfileirado",
  "POST commands": "Comando enfileirado",
  "PUT policy": "Política atualizada",
  "PUT routines": "Rotina atualizada",
  "POST routines": "Rotina criada",
  "DELETE routines": "Rotina excluída",
};
const retentionOptions = [1, 7, 15, 30, 60, 90];

function operationName(value: string) { return operationNames[value] ?? value.replaceAll("_", " "); }
function localTime(value: string) {
  const date = new Date(value);
  return `${date.toLocaleTimeString("pt-BR", { hour12: false })}.${String(date.getMilliseconds()).padStart(3, "0")}`;
}
function resultText(event: CommunicationLog) {
  if (event.http_status) {
    if (event.result === "error") return `${event.http_status} Falhou`;
    if (event.result === "warning") return `${event.http_status} Atenção`;
    return `${event.http_status} OK`;
  }
  return event.result === "success" ? "Concluído" : event.result === "warning" ? "Atenção" : "Falhou";
}
function routeText(event: CommunicationLog) { return `${partyNames[event.source]} → ${partyNames[event.target]}`; }
const commandNames: Record<string, string> = {
  pause_monitoring: "pausa",
  resume_monitoring: "retomada",
  block_now: "bloqueio",
  clear_manual_block: "desbloqueio",
};
function friendlyCommand(value: string | undefined) { return value ? commandNames[value] ?? value.replaceAll("_", " ") : ""; }
function describeEvent(event: CommunicationLog, deviceName: string): string {
  const machine = deviceName || "o computador";
  const { source, target, operation, details } = event;
  if (source === "agent") {
    const action = operation === "heartbeat" ? "enviou atualização de estado" : `enviou ${operationName(operation).toLocaleLowerCase("pt-BR")}`;
    return `O computador ${machine} ${action} à API`;
  }
  if (source === "api") {
    const action = operationName(operation).toLocaleLowerCase("pt-BR");
    return target === "agent" ? `A API enviou ${action} ao computador ${machine}` : `A API respondeu ao administrador: ${operationName(operation)}`;
  }
  switch (operation) {
    case "POST bonus": return `O administrador concedeu ${details.bonus_minutes ? `${details.bonus_minutes} min` : "tempo extra"} de bônus a ${machine}`;
    case "POST commands": return `O administrador enviou o comando de ${friendlyCommand(details.command) || "controle"} para ${machine}`;
    case "PUT policy": return `O administrador atualizou a política de ${machine}${details.warning_minutes ? ` (alerta após ${details.warning_minutes} min)` : ""}`;
    case "POST routines": return `O administrador criou a rotina "${details.routine_name ?? "—"}" em ${machine}`;
    case "PUT routines": return `O administrador alterou a rotina "${details.routine_name ?? "—"}" em ${machine}`;
    case "DELETE routines": return `O administrador excluiu uma rotina de ${machine}`;
    case "PUT password": return `O administrador redefiniu a senha local de ${machine}`;
    case "POST token": return `O administrador emitiu um novo token de acesso para ${machine}`;
    case "DELETE token": return `O administrador revogou o token de acesso de ${machine}`;
    case "PATCH device": return `O administrador renomeou ${machine} para "${details.device_name ?? "—"}"`;
    case "DELETE device": return `O administrador removeu o computador ${machine}`;
    default: return `O administrador ${operationName(operation).toLocaleLowerCase("pt-BR")}`;
  }
}
function mergeEvents(current: CommunicationLog[], incoming: CommunicationLog[]) {
  const known = new Set(current.map((event) => event.id));
  return [...incoming.filter((event) => !known.has(event.id)).reverse(), ...current].slice(0, 500);
}

export function CommunicationPage({ deviceId, deviceName }: { deviceId: string; deviceName: string }) {
  const [events, setEvents] = useState<CommunicationLog[]>(remoteMode ? [] : mockCommunication);
  const [retentionDays, setRetentionDays] = useState(30);
  const [search, setSearch] = useState("");
  const [party, setParty] = useState<PartyFilter>("all");
  const [result, setResult] = useState<ResultFilter>("all");
  const [hideHeartbeats, setHideHeartbeats] = useState(false);
  const [expanded, setExpanded] = useState<number | null>(null);
  const [liveState, setLiveState] = useState<LiveState>(remoteMode ? "connecting" : "live");
  const [error, setError] = useState("");
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [busy, setBusy] = useState(false);
  const maxID = useRef(0);
  const polling = useRef(false);
  const generation = useRef(0);

  useEffect(() => {
    generation.current += 1;
    setExpanded(null);
    setError("");
    if (!remoteMode) {
      setEvents(mockCommunication.map((event) => ({ ...event, device_id: deviceId })));
      setLiveState("live");
      return;
    }
    let active = true;
    maxID.current = 0;
    setEvents([]);
    setLiveState("connecting");

    const refresh = async (initial = false) => {
      if (polling.current) return;
      polling.current = true;
      const requestGeneration = generation.current;
      try {
        const response = await api.communication(deviceId, initial ? 0 : maxID.current);
        if (!active || requestGeneration !== generation.current) return;
        setRetentionDays(response.retention_days);
        if (response.events.length) {
          maxID.current = Math.max(maxID.current, ...response.events.map((event) => event.id));
          setEvents((current) => initial ? response.events : mergeEvents(current, response.events));
        }
        setError("");
        setLiveState("live");
      } catch (loadError) {
        if (!active) return;
        setError(loadError instanceof Error ? loadError.message : "Não foi possível atualizar os registros.");
        setLiveState("interrupted");
      } finally {
        polling.current = false;
      }
    };

    let stream: EventSource | null = null;
    let connected = false;
    const onStreamEvent = (event: MessageEvent) => {
      if (!active) return;
      try {
        const log = JSON.parse(event.data) as CommunicationLog;
        setEvents((current) => mergeEvents(current, [log]));
        if (log.id > maxID.current) maxID.current = log.id;
        setError("");
        setLiveState("live");
      } catch { /* ignora evento inválido */ }
    };
    stream = api.openStream(deviceId);
    stream.addEventListener("communication", onStreamEvent);
    stream.onopen = () => {
      if (!active) return;
      connected = true;
      setLiveState("live");
      void refresh();
    };
    stream.onerror = () => {
      if (!active) return;
      connected = false;
      setLiveState("interrupted");
    };

    void refresh(true);
    const interval = window.setInterval(() => { if (!connected) void refresh(); }, 1000);
    return () => {
      active = false;
      generation.current += 1;
      if (stream) stream.close();
      window.clearInterval(interval);
      polling.current = false;
    };
  }, [deviceId]);

  const visible = useMemo(() => {
    const term = search.trim().toLocaleLowerCase("pt-BR");
    return events.filter((event) => {
      if (hideHeartbeats && event.operation === "heartbeat") return false;
      if (party !== "all" && event.source !== party && event.target !== party) return false;
      if (result !== "all" && event.result !== result) return false;
      if (!term) return true;
      const searchable = [routeText(event), operationName(event.operation), describeEvent(event, deviceName), event.summary, ...Object.values(event.details)].join(" ").toLocaleLowerCase("pt-BR");
      return searchable.includes(term);
    });
  }, [events, party, result, search, hideHeartbeats, deviceName]);

  const saveRetention = async (days: number) => {
    const previous = retentionDays;
    setRetentionDays(days);
    try {
      if (remoteMode) {
        generation.current += 1;
        await api.setCommunicationRetention(deviceId, days);
        const refreshed = await api.communication(deviceId);
        setEvents(refreshed.events);
        maxID.current = refreshed.events.length ? Math.max(...refreshed.events.map((event) => event.id)) : 0;
      }
    } catch (saveError) {
      setRetentionDays(previous);
      setError(saveError instanceof Error ? saveError.message : "Não foi possível salvar a retenção.");
    }
  };

  const deleteLogs = async () => {
    setBusy(true);
    generation.current += 1;
    try {
      if (remoteMode) await api.deleteCommunication(deviceId);
      setEvents([]);
      maxID.current = 0;
      setExpanded(null);
      setConfirmDelete(false);
    } catch (deleteError) {
      setError(deleteError instanceof Error ? deleteError.message : "Não foi possível excluir os registros.");
    } finally {
      setBusy(false);
    }
  };

  return <section className="communication-page">
    <header className="communication-heading">
      <div><h2>Comunicação</h2><div className={`live-indicator ${liveState}`}><span aria-hidden="true" />{liveState === "live" ? "Atualização ao vivo" : liveState === "connecting" ? "Conectando…" : "Atualização interrompida"}<i /> <small>Novos eventos aparecem automaticamente</small></div></div>
    </header>

    <div className="communication-controls">
      <label className="log-search"><span className="sr-only">Buscar nos registros</span><Search aria-hidden="true" size={18} /><input placeholder="Buscar nos registros" value={search} onChange={(event) => setSearch(event.target.value)} /></label>
      <label><span className="sr-only">Participante</span><select value={party} onChange={(event) => setParty(event.target.value as PartyFilter)}><option value="all">Todos</option><option value="agent">Agente</option><option value="api">API</option><option value="interface">Interface</option></select></label>
      <label><span className="sr-only">Resultado</span><select value={result} onChange={(event) => setResult(event.target.value as ResultFilter)}><option value="all">Todos os resultados</option><option value="success">Concluídos</option><option value="warning">Com atenção</option><option value="error">Com falha</option></select></label>
      <label className="heartbeat-toggle"><input type="checkbox" checked={hideHeartbeats} onChange={(event) => setHideHeartbeats(event.target.checked)} />Ocultar batimentos</label>
      <label><span className="sr-only">Retenção dos registros</span><select aria-label="Retenção dos registros" value={retentionDays} onChange={(event) => void saveRetention(Number(event.target.value))}>{retentionOptions.map((days) => <option key={days} value={days}>Manter por {days} {days === 1 ? "dia" : "dias"}</option>)}</select></label>
      <button className="delete-logs" onClick={() => setConfirmDelete(true)}><Trash2 size={17} />Excluir logs</button>
    </div>

    {error && <div className="communication-error" role="alert"><AlertTriangle size={19} /><span><strong>Atualização incompleta</strong>{error}</span></div>}

    <div className="log-table" role="table" aria-label={`Comunicação de ${deviceName}`}>
      <div className="log-table-header" role="row"><span>Hora local</span><span>O que aconteceu</span><span>Resultado</span><span /></div>
      {visible.map((event) => <div className={`log-entry ${event.result}`} key={event.id}>
        <button aria-expanded={expanded === event.id} className="log-row" onClick={() => setExpanded((current) => current === event.id ? null : event.id)} role="row">
          <time dateTime={event.created_at}>{localTime(event.created_at)}</time>
          <span className="log-message"><strong>{describeEvent(event, deviceName)}</strong><small>{operationName(event.operation)}</small></span>
          <b className={`log-result ${event.result}`}>{resultText(event)}</b>
          {expanded === event.id ? <ChevronDown aria-hidden="true" size={18} /> : <ChevronRight aria-hidden="true" size={18} />}
        </button>
        {expanded === event.id && <div className="log-details"><dl>{Object.entries(event.details).map(([key, value]) => <div key={key}><dt>{key}</dt><dd>{value || "—"}</dd></div>)}<div><dt>Status HTTP</dt><dd>{event.http_status ?? "—"}</dd></div><div><dt>Duração</dt><dd>{event.duration_ms ? `${event.duration_ms.toLocaleString("pt-BR")} ms` : "—"}</dd></div></dl><p>Campos sensíveis foram omitidos.</p></div>}
      </div>)}
      {!visible.length && <div className="log-empty"><Radio size={24} /><strong>{events.length ? "Nenhum registro corresponde aos filtros" : "Aguardando comunicação"}</strong><span>{events.length ? "Ajuste a busca ou os filtros." : "Os próximos eventos aparecerão aqui automaticamente."}</span></div>}
    </div>
    <footer className="communication-footer">Mostrando {visible.length} de {events.length} registros mantidos nesta sessão</footer>

    {confirmDelete && <Modal title="Excluir logs?" description={`Todos os registros de comunicação de “${deviceName}” serão removidos.`} onClose={() => !busy && setConfirmDelete(false)}><div className="delete-log-warning"><AlertTriangle size={21} /><p>Essa ação não altera o histórico de uso, bônus ou configurações do computador.</p></div><div className="modal-actions"><button disabled={busy} onClick={() => setConfirmDelete(false)}>Cancelar</button><button className="danger-confirm" disabled={busy} onClick={() => void deleteLogs()}><Trash2 size={17} />{busy ? "Excluindo…" : "Excluir logs"}</button></div></Modal>}
  </section>;
}
