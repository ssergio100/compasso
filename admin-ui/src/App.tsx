import { Activity, AlertTriangle, CalendarDays, Check, ChevronDown, Clock3, Copy, KeyRound, Laptop, LockKeyhole, LogOut, Monitor, MonitorCheck, Pause, Pencil, Play, Plus, RefreshCw, Save, Settings, ShieldCheck, ShieldOff, SlidersHorizontal, SquareTerminal, Trash2, UnlockKeyhole, UserRoundCheck, X } from "lucide-react";
import { useEffect, useMemo, useRef, useState, type FormEvent, type ReactNode } from "react";
import { api, remoteMode } from "./api";
import { Brand, DurationWheel, Modal, TimeRangePicker, Toast } from "./components";
import { CommunicationPage } from "./communication/CommunicationPage";
import { mockDevices } from "./mock";
import type { AvatarKey, CommunicationLog, Device, DeviceActivity, LiveStatus, LiveStreamState, Routine, RoutineIconKey, View } from "./types";
import { AvatarPicker, defaultAvatarKey, DeviceAvatar, inferRoutineIcon, RoutineIconPicker, RoutineVisual } from "./visuals";

const dayNames = ["Dom", "Seg", "Ter", "Qua", "Qui", "Sex", "Sáb"];
const nav: { id: View; label: string; short?: string; Icon: typeof Activity }[] = [
  { id: "now", label: "Agora", Icon: Activity }, { id: "limits", label: "Limites", Icon: SlidersHorizontal },
  { id: "routines", label: "Rotinas", Icon: CalendarDays }, { id: "administration", label: "Administração", short: "Admin.", Icon: Settings },
  { id: "communication", label: "Atividade", short: "Ativ.", Icon: SquareTerminal },
];

function fmt(seconds: number) {
  const total = Math.max(0, Math.round(seconds / 60)); const hours = Math.floor(total / 60); const minutes = total % 60;
  if (!hours) return `${minutes}min`; if (!minutes) return `${hours}h`; return `${hours}h ${String(minutes).padStart(2, "0")}min`;
}
function clock(seconds: number) { const total = Math.floor(seconds / 60); return `${String(Math.floor(total / 60) % 24).padStart(2, "0")}:${String(total % 60).padStart(2, "0")}`; }
function seconds(value: string) { const [h, m] = value.split(":").map(Number); return h * 3600 + m * 60; }
function lastSeen(value: string | null) { if (!value) return "Ainda não sincronizado"; const minutes = Math.round((Date.now() - new Date(value).getTime()) / 60000); return minutes < 2 ? "Agora mesmo" : `Há ${minutes} minutos`; }
function localID() { return typeof globalThis.crypto?.randomUUID === "function" ? globalThis.crypto.randomUUID() : `local-${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`; }
function deviceVisualState(device: Device) { if (!device.online) return "offline"; if (device.actual_state === "blocked") return "blocked"; if (["block_requested", "pause_requested"].includes(device.control_status)) return "pending"; if (device.control_status === "paused") return "paused"; return "online"; }
function avatarKeyFor(device: Device): AvatarKey { return device.avatar_key ?? defaultAvatarKey(device.id); }
function routineIconFor(routine: Routine): RoutineIconKey { return routine.icon_key ?? inferRoutineIcon(routine.name); }
function DeviceState({ device }: { device: Device }) { const state = deviceVisualState(device); const Icon = state === "blocked" ? LockKeyhole : state === "paused" ? Pause : state === "pending" ? RefreshCw : state === "online" ? MonitorCheck : ShieldOff; const label = state === "blocked" ? "Bloqueado" : state === "paused" ? "Pausado" : device.control_status === "block_requested" ? "Bloqueando…" : device.control_status === "pause_requested" ? "Pausando…" : state === "online" ? "Online" : "Offline"; return <small className={`device-state ${state}`}><Icon aria-hidden="true" size={13} />{label}</small>; }

async function copyVisibleValue(value: string) {
  if (window.isSecureContext && navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(value);
      return;
    } catch { /* usa a cópia compatível abaixo */ }
  }
  const previouslyFocused = document.activeElement instanceof HTMLElement ? document.activeElement : null;
  const temporaryField = document.createElement("textarea");
  temporaryField.value = value;
  temporaryField.readOnly = true;
  temporaryField.setAttribute("aria-hidden", "true");
  temporaryField.style.cssText = "position:fixed;inset:0 auto auto -10000px;opacity:0;pointer-events:none";
  document.body.appendChild(temporaryField);
  temporaryField.select();
  temporaryField.setSelectionRange(0, value.length);
  let copied = false;
  try {
    copied = document.execCommand("copy");
  } finally {
    temporaryField.remove();
    previouslyFocused?.focus();
  }
  if (!copied) throw new Error("copy unavailable");
}

export function App() {
  const [devices, setDevices] = useState<Device[]>(remoteMode ? [] : mockDevices);
  const [selectedId, setSelectedId] = useState(mockDevices[0].id);
  const [view, setView] = useState<View>("now");
  const [authenticated, setAuthenticated] = useState(!remoteMode);
  const [checking, setChecking] = useState(remoteMode);
  const [loading, setLoading] = useState(remoteMode);
  const [message, setMessage] = useState("");
  const [modal, setModal] = useState<"bonus" | "device" | "routine" | null>(null);
  const [editingRoutine, setEditingRoutine] = useState<Routine | null>(null);
  const [streamState, setStreamState] = useState<LiveStreamState>(remoteMode ? "connecting" : "live");
  const [streamGeneration, setStreamGeneration] = useState(0);
  const [activityUpdate, setActivityUpdate] = useState<DeviceActivity | null>(null);
  const [communicationUpdate, setCommunicationUpdate] = useState<CommunicationLog | null>(null);
  const pendingOperations = useRef(new Set<string>());
  const selected = useMemo(() => devices.find((item) => item.id === selectedId) ?? devices[0], [devices, selectedId]);

  const notify = (text: string) => { setMessage(text); window.setTimeout(() => setMessage(""), 2600); };
  const load = async () => { setLoading(true); try { const list = await api.devices(); setDevices(list); if (list.length) setSelectedId((id) => list.some((d) => d.id === id) ? id : list[0].id); } catch (error) { notify(error instanceof Error ? error.message : "Falha ao carregar dados."); } finally { setLoading(false); } };
  useEffect(() => { if (!remoteMode) return; api.session().then((session) => { setAuthenticated(session.authenticated); if (session.authenticated) void load(); }).catch(() => setAuthenticated(false)).finally(() => setChecking(false)); }, []);
  const applyLiveStatus = (status: LiveStatus) => setDevices((all) => all.map((item) => item.id === selected.id ? { ...item, online: status.online, graphical_session_active: status.graphical_session_active, actual_state: status.actual_state, control_status: status.control_status, counting: status.counting, used_seconds: status.used_seconds, remaining_seconds: status.remaining_seconds, bonus_seconds: status.bonus_seconds, today_quota_seconds: status.today_quota_seconds, last_seen_at: status.online ? new Date().toISOString() : item.last_seen_at } : item));
  useEffect(() => {
    if (!remoteMode || !selected) {
      setStreamState("live");
      return;
    }
    setStreamState("connecting");
    setActivityUpdate(null);
    setCommunicationUpdate(null);
    const stream = api.openStream(selected.id);
    const onStatus = (event: MessageEvent) => { try { applyLiveStatus(JSON.parse(event.data) as LiveStatus); } catch { /* ignora evento inválido */ } };
    stream.addEventListener("hello", onStatus);
    stream.addEventListener("status", onStatus);
    stream.addEventListener("device_offline", onStatus);
    const onActivity = (event: MessageEvent) => {
      try {
        const activity = JSON.parse(event.data) as DeviceActivity;
        setActivityUpdate(activity);
        if (activity.status === "completed" && pendingOperations.current.delete(activity.id)) {
          notify(activity.kind === "add_bonus" ? "Tempo extra aplicado e confirmado pelo computador." : "Alteração aplicada e confirmada pelo computador.");
        }
      } catch { /* ignora evento inválido */ }
    };
    stream.addEventListener("activity_updated", onActivity);
	stream.addEventListener("activities_changed", () => setStreamGeneration((generation) => generation + 1));
    stream.addEventListener("communication", (event) => {
      try { setCommunicationUpdate(JSON.parse((event as MessageEvent).data) as CommunicationLog); } catch { /* ignora evento inválido */ }
    });
    stream.onopen = () => {
      setStreamState("live");
      setStreamGeneration((generation) => generation + 1);
    };
    stream.onerror = () => setStreamState("interrupted");
    return () => stream.close();
  }, [selected?.id, remoteMode]);

  const patchDevice = (change: (device: Device) => Device) => setDevices((all) => all.map((item) => item.id === selected.id ? change(item) : item));
  const command = async (name: string, local: (device: Device) => Device, success: string) => { try { if (remoteMode) { const confirmation = await api.command(selected.id, name); pendingOperations.current.add(confirmation.operation_id); } patchDevice(local); notify(success); if (remoteMode) await load(); } catch (error) { notify(error instanceof Error ? error.message : "Comando não enviado."); } };

  if (checking) return <div className="center-state"><Brand /><span className="loader" />Verificando sessão…</div>;
  if (!authenticated) return <Login onLogin={async (login, password) => { const session = await api.login(login, password); setAuthenticated(session.authenticated); if (session.authenticated) await load(); return session.authenticated; }} />;
  if (loading && !selected) return <div className="center-state"><Brand /><span className="loader" />Organizando seus computadores…</div>;
  if (!selected) return <div className="center-state"><Brand /><h1>Nenhum computador</h1><button className="primary-button" onClick={() => setModal("device")}><Plus size={18} />Adicionar computador</button>{modal === "device" && <DeviceModal onClose={() => setModal(null)} onSubmit={async (name, avatarKey) => { if (remoteMode) await api.createDevice(name, avatarKey); else setDevices([{ ...mockDevices[1], id: localID(), name, avatar_key: avatarKey }]); setModal(null); if (remoteMode) await load(); }} />}{message && <Toast>{message}</Toast>}</div>;

  return <div className="app-shell">
    <a className="skip" href="#workspace">Pular para o conteúdo</a>
    <aside className="device-rail"><Brand /><div className="rail-title">Computadores <button aria-label="Adicionar computador" onClick={() => setModal("device")}><Plus size={18} /></button></div>{devices.map((device) => <button className={`${device.id === selected.id ? "active " : ""}device-${deviceVisualState(device)}`} key={device.id} onClick={() => { setSelectedId(device.id); setView("now"); }}><DeviceAvatar avatarKey={avatarKeyFor(device)} name={device.name} /><span><strong>{device.name}</strong><DeviceState device={device} /></span></button>)}<button className="add-device" onClick={() => setModal("device")}><Plus size={17} />Adicionar computador</button><div className="rail-account"><UserRoundCheck size={19} /><span><small>Sessão</small><strong>Administrador</strong></span><button aria-label="Sair" onClick={async () => { if (remoteMode) await api.logout(); setAuthenticated(false); }}><LogOut size={18} />Sair</button></div></aside>
    <aside className="main-nav"><div className="nav-title">Ações</div><nav>{nav.map(({ id, label, Icon }) => <button className={view === id ? "active" : ""} key={id} onClick={() => setView(id)}><Icon size={20} />{label}</button>)}</nav></aside>
    <main id="workspace">
      <header className="mobile-header"><div className="mobile-account"><Brand /><button onClick={async () => { if (remoteMode) await api.logout(); setAuthenticated(false); }}><LogOut size={17} />Sair</button></div><details><summary><DeviceAvatar avatarKey={avatarKeyFor(selected)} name={selected.name} /><span><strong>{selected.name}</strong><DeviceState device={selected} /></span><ChevronDown size={19} /></summary><div>{devices.map((device) => <button key={device.id} onClick={() => setSelectedId(device.id)}><DeviceAvatar avatarKey={avatarKeyFor(device)} name={device.name} /><span>{device.name}<DeviceState device={device} /></span></button>)}</div></details></header>
      <header className="workspace-header"><div><DeviceAvatar avatarKey={avatarKeyFor(selected)} className="workspace-avatar" name={selected.name} /><span><h1>{selected.name}</h1><DeviceState device={selected} /></span></div><button onClick={() => void load()}><RefreshCw size={17} />{lastSeen(selected.last_seen_at)}</button></header>
      <div className={`workspace-body ${view === "communication" ? "communication-workspace" : ""}`}>
        {view === "now" && <Now device={selected} onBonus={() => setModal("bonus")} onPause={() => void command(selected.monitoring_paused ? "resume_monitoring" : "pause_monitoring", (d) => ({ ...d, monitoring_paused: !d.monitoring_paused, control_status: d.monitoring_paused ? "active" : remoteMode ? "pause_requested" : "paused" }), selected.monitoring_paused ? "Pausa cancelada." : "Pausa solicitada.")} onBlock={() => void command(selected.manual_block ? "clear_manual_block" : "block_now", (d) => ({ ...d, monitoring_paused: d.manual_block ? d.monitoring_paused : false, manual_block: !d.manual_block, actual_state: d.manual_block ? remoteMode ? d.actual_state : "unblocked" : remoteMode ? d.actual_state : "blocked", control_status: d.manual_block ? "active" : remoteMode ? "block_requested" : "blocked" }), selected.manual_block ? "Remoção do bloqueio solicitada." : "Bloqueio solicitado.")} />}
        {view === "limits" && <Limits device={selected} onSave={async (weekly) => { if (remoteMode) await api.policy(selected.id, weekly, selected.warning_minutes); patchDevice((d) => ({ ...d, weekly_quota_seconds: weekly })); notify("Limites salvos."); if (remoteMode) await load(); }} />}
        {view === "routines" && <Routines device={selected} onNew={() => { setEditingRoutine(null); setModal("routine"); }} onEdit={(routine) => { setEditingRoutine(routine); setModal("routine"); }} onToggle={async (routine) => { const next = { ...routine, enabled: !routine.enabled }; if (remoteMode) await api.routine(selected.id, withoutId(next), routine.id); patchDevice((d) => ({ ...d, routines: d.routines.map((r) => r.id === routine.id ? next : r) })); notify(next.enabled ? "Rotina ativada." : "Rotina pausada."); }} onDelete={async (routine) => { if (remoteMode) await api.deleteRoutine(selected.id, routine.id); patchDevice((d) => ({ ...d, routines: d.routines.filter((r) => r.id !== routine.id) })); notify("Rotina removida."); }} />}
        {view === "administration" && <Administration device={selected} onSave={async (name, warning, avatarKey) => { if (remoteMode) { await api.rename(selected.id, name, avatarKey); await api.policy(selected.id, selected.weekly_quota_seconds, warning); await load(); } patchDevice((d) => ({ ...d, name, warning_minutes: warning, avatar_key: avatarKey })); notify("Informações salvas."); }} onPassword={async (password, confirmation) => { if (remoteMode) await api.updatePassword(selected.id, password, confirmation); patchDevice((d) => ({ ...d, password_set: true })); notify("Senha local atualizada."); if (remoteMode) await load(); }} onIssueToken={async () => { const result = remoteMode ? await api.issueToken(selected.id) : { device_id: selected.id, device_token: mockToken() }; notify("Novo token gerado."); return result.device_token; }} onRevokeToken={async () => { if (remoteMode) await api.revokeToken(selected.id); notify("Token revogado."); }} onDelete={async () => { if (remoteMode) await api.deleteDevice(selected.id); const remaining = devices.filter((device) => device.id !== selected.id); setDevices(remaining); setSelectedId(remaining[0]?.id ?? ""); setView("now"); notify("Computador excluído."); }} />}
        {view === "communication" && <CommunicationPage
          activityUpdate={activityUpdate}
          communicationUpdate={communicationUpdate}
          deviceId={selected.id}
          deviceName={selected.name}
          streamGeneration={streamGeneration}
          streamState={streamState}
        />}
      </div>
    </main>
    <nav className="bottom-nav">{nav.map(({ id, label, short, Icon }) => <button className={view === id ? "active" : ""} key={id} onClick={() => setView(id)}><Icon size={21} /><span>{short ?? label}</span></button>)}</nav>
    {modal === "bonus" && <BonusModal onClose={() => setModal(null)} onSubmit={async (minutes) => { if (!remoteMode) { patchDevice((d) => ({ ...d, bonus_seconds: d.bonus_seconds + minutes * 60, remaining_seconds: d.remaining_seconds + minutes * 60 })); setModal(null); notify(`${minutes} minutos adicionados.`); return; } const confirmation = await api.bonus(selected.id, minutes); pendingOperations.current.add(confirmation.operation_id); setModal(null); setMessage("Pedido guardado pelo servidor. Aguardando o computador confirmar."); }} />}
    {modal === "device" && <DeviceModal onClose={() => setModal(null)} onSubmit={async (name, avatarKey) => { if (remoteMode) { const created = await api.createDevice(name, avatarKey); setSelectedId(created.id); await load(); } else { const device = { ...mockDevices[1], id: localID(), name, avatar_key: avatarKey }; setDevices((all) => [...all, device]); setSelectedId(device.id); } setModal(null); notify("Computador adicionado."); }} />}
    {modal === "routine" && <RoutineModal initial={editingRoutine ?? undefined} routines={selected.routines.filter((routine) => routine.id !== editingRoutine?.id)} onClose={() => { setModal(null); setEditingRoutine(null); }} onSubmit={async (draft) => { const routineId = editingRoutine?.id; const saved = remoteMode ? await api.routine(selected.id, draft, routineId) : { id: routineId ?? localID() }; const savedId = routineId ?? saved.id; if (remoteMode) await load(); patchDevice((d) => ({ ...d, routines: d.routines.some((routine) => routine.id === savedId) ? d.routines.map((routine) => routine.id === savedId ? { ...draft, id: savedId } : routine) : [...d.routines, { ...draft, id: savedId || localID() }] })); setModal(null); setEditingRoutine(null); notify(routineId ? "Rotina atualizada." : "Rotina criada."); }} />}
    {message && <Toast>{message}</Toast>}
  </div>;
}

function withoutId(routine: Routine): Omit<Routine, "id"> { const { id: _, ...value } = routine; return value; }

function routineIntervals(routine: Omit<Routine, "id">): [number, number][] { const daySeconds = 86400; const intervals: [number, number][] = []; for (let day = 0; day < 7; day += 1) { const previous = (day + 6) % 7; if (routine.start_second === routine.end_second) { if (routine.days[day]) intervals.push([day * daySeconds, (day + 1) * daySeconds]); } else if (routine.start_second < routine.end_second) { if (routine.days[day]) intervals.push([day * daySeconds + routine.start_second, day * daySeconds + routine.end_second]); } else { if (routine.days[previous] && routine.end_second > 0) intervals.push([day * daySeconds, day * daySeconds + routine.end_second]); if (routine.days[day]) intervals.push([day * daySeconds + routine.start_second, (day + 1) * daySeconds]); } } return intervals; }
function conflictingRoutine(draft: Omit<Routine, "id">, routines: Routine[]): Routine | undefined { const intervals = routineIntervals(draft); return routines.find((routine) => routineIntervals(routine).some(([start, end]) => intervals.some(([draftStart, draftEnd]) => start < draftEnd && draftStart < end))); }

function Login({ onLogin }: { onLogin: (login: string, password: string) => Promise<boolean> }) {
  const [login, setLogin] = useState(""); const [password, setPassword] = useState(""); const [error, setError] = useState(""); const [busy, setBusy] = useState(false);
  return <main className="login"><section><Brand /><h1>Acesso administrativo</h1><p>Administre limites e rotinas.</p><form onSubmit={async (event) => { event.preventDefault(); setBusy(true); try { if (!await onLogin(login, password)) setError("Usuário ou senha inválidos."); } catch { setError("Não foi possível entrar."); } finally { setBusy(false); } }}><label>Usuário<input autoComplete="username" value={login} onChange={(event) => setLogin(event.target.value)} /></label><label>Senha<input autoComplete="current-password" type="password" value={password} onChange={(event) => setPassword(event.target.value)} /></label>{error && <p className="error">{error}</p>}<button className="primary-button" disabled={busy || !login || password.length < 6}>{busy ? "Entrando…" : "Entrar"}</button></form></section></main>;
}

interface RoutineSegment { key: string; name: string; start: number; end: number }

function nextRoutineLabel(routines: Routine[], now: Date) {
  let next: { routine: Routine; at: Date; offset: number } | undefined;
  for (let offset = 0; offset <= 7; offset += 1) {
    for (const routine of routines) {
      if (!routine.enabled) continue;
      const at = new Date(now);
      at.setDate(now.getDate() + offset);
      if (!routine.days[at.getDay()]) continue;
      const start = routine.start_second === routine.end_second ? 0 : routine.start_second;
      at.setHours(Math.floor(start / 3600), Math.floor(start / 60) % 60, start % 60, 0);
      if (at <= now || next && at >= next.at) continue;
      next = { routine, at, offset };
    }
  }
  if (!next) return "Nenhuma programada";
  const day = next.offset === 0 ? "Hoje" : next.offset === 1 ? "Amanhã" : dayNames[next.at.getDay()];
  return `${next.routine.name} · ${day}, ${clock(next.routine.start_second === next.routine.end_second ? 0 : next.routine.start_second)}`;
}

function routineSegmentsForDay(routines: Routine[], day: number): RoutineSegment[] {
  const previousDay = (day + 6) % 7;
  return routines.flatMap((routine) => {
    if (!routine.enabled) return [];
    const segment = (start: number, end: number, edge: string): RoutineSegment => ({
      key: `${routine.id}-${edge}`, name: routine.name, start, end,
    });
    if (routine.start_second === routine.end_second) {
      return routine.days[day] ? [segment(0, 86400, "full")] : [];
    }
    if (routine.start_second < routine.end_second) {
      return routine.days[day] ? [segment(routine.start_second, routine.end_second, "day")] : [];
    }
    const overnight: RoutineSegment[] = [];
    if (routine.days[previousDay] && routine.end_second > 0) overnight.push(segment(0, routine.end_second, "morning"));
    if (routine.days[day]) overnight.push(segment(routine.start_second, 86400, "night"));
    return overnight;
  });
}

function Now({ device, onBonus, onPause, onBlock }: { device: Device; onBonus: () => void; onPause: () => void; onBlock: () => void }) {
  const remaining = device.remaining_seconds;
  const [now, setNow] = useState(() => new Date());
  useEffect(() => { const timer = window.setInterval(() => setNow(new Date()), 30000); return () => window.clearInterval(timer); }, []);
  const minutes = now.getHours() * 60 + now.getMinutes(); const daySegments = [-1, 0, 1].map((offset) => ({ offset, when: ["ontem", "hoje", "amanhã"][offset + 1], segments: routineSegmentsForDay(device.routines, (now.getDay() + offset + 7) % 7) }));
  const visualState = deviceVisualState(device);
  const blockPending = device.control_status === "block_requested";
  const pausePending = device.control_status === "pause_requested";
  const monitorState = device.control_status === "pause_requested" ? "Pausa solicitada" : device.control_status === "paused" ? "Pausado" : !device.graphical_session_active ? "Aguardando sessão" : "Ativo";
  const accessState = device.actual_state === "blocked" ? "Bloqueado" : blockPending ? "Bloqueio solicitado" : !device.graphical_session_active ? "Sem sessão" : "Liberado";
  return <section className={`rhythm-board state-${visualState}`}><h2 className="section-name">Agora</h2><div className="time-hero"><strong>{fmt(remaining)}</strong><span>{!device.online ? "saldo calculado com os últimos dados" : visualState === "blocked" || visualState === "paused" ? "tempo restante preservado" : "restantes hoje"}</span></div><div className="day-track"><div className="day-window"><div className="day-strip" style={{ transform: `translateX(${(-(minutes / 1440) * 100 - 50) / 3}%)` }}>{daySegments.map(({ offset, when, segments }) => <div className={offset === 0 ? "day-cell" : "day-cell adjacent"} key={offset}><div className="track-labels"><span>00:00</span><span>06:00</span><b>12:00</b><span>18:00</span><span /></div>{segments.length > 0 && <div aria-label={`Rotinas de bloqueio de ${when}`} className="routine-bands">{segments.map((segment) => { const start = clock(segment.start); const end = segment.end === 86400 ? "24:00" : clock(segment.end); const description = `${segment.name}: bloqueio das ${start} às ${end}`; return <span aria-label={`${description} de ${when}`} key={segment.key} role="img" style={{ left: `${(segment.start / 86400) * 100}%`, width: `${((segment.end - segment.start) / 86400) * 100}%` }} title={description} />; })}</div>}<div className="ticks" /></div>)}</div></div><i aria-hidden="true" /></div>{daySegments.some(({ segments }) => segments.length > 0) && <div className="track-legend"><span aria-hidden="true" /><LockKeyhole size={13} />Rotinas de bloqueio</div>}<dl className="time-facts"><div><dt>{device.online ? "Usado" : "Uso registrado"}</dt><dd>{fmt(device.used_seconds)}</dd></div><div><dt>Limite de hoje</dt><dd>{fmt(device.today_quota_seconds + device.bonus_seconds)}</dd></div></dl><div className="quick-actions"><button className="primary" onClick={onBonus}><Plus size={20} />Mais tempo</button><button className={device.control_status === "paused" ? "state-action paused" : pausePending ? "state-action pending" : ""} disabled={!device.online || blockPending} onClick={onPause} title={!device.online ? "Disponível quando o computador estiver conectado" : pausePending ? "Cancela a pausa solicitada" : undefined}>{pausePending ? <Play size={20} /> : device.monitoring_paused ? <Play size={20} /> : <Pause size={20} />}{pausePending ? "Cancelar pausa" : device.monitoring_paused ? "Retomar" : "Pausar"}</button><button className={device.actual_state === "blocked" ? "state-action blocked" : blockPending ? "state-action pending" : ""} disabled={!device.online || pausePending} onClick={onBlock} title={!device.online ? "Disponível quando o computador estiver conectado" : blockPending ? "Cancela o bloqueio solicitado" : undefined}>{device.manual_block ? <UnlockKeyhole size={20} /> : <LockKeyhole size={20} />}{blockPending ? device.actual_state === "blocked" ? "Remover bloqueio" : "Cancelar bloqueio" : device.manual_block ? "Remover bloqueio" : device.actual_state === "blocked" ? "Manter bloqueado" : "Bloquear"}</button></div><div className={`detail-grid ${device.online ? "" : "offline-summary"}`}><DetailSection title="Resumo de hoje" Icon={CalendarDays}><Row Icon={Clock3} label="Próxima rotina" value={nextRoutineLabel(device.routines, now)} />{device.online && <Row Icon={RefreshCw} label="Última sincronização" value={lastSeen(device.last_seen_at)} />}</DetailSection>{device.online && <DetailSection title="Estado do computador" Icon={Monitor}><Row Icon={ShieldCheck} label="Agente" value="Conectado" /><Row Icon={UserRoundCheck} label="Sessão gráfica" value={device.graphical_session_active ? "Ativa" : "Sem sessão"} /><Row Icon={MonitorCheck} label="Monitoramento" value={monitorState} /><Row Icon={LockKeyhole} label="Acesso" value={accessState} /><Row Icon={Clock3} label="Contagem de tempo" value={device.counting ? "Em andamento" : "Parada"} /></DetailSection>}</div></section>;
}
function DetailSection({ title, Icon, children }: { title: string; Icon: typeof Clock3; children: ReactNode }) { return <section className="detail-section"><h3><Icon size={21} />{title}</h3>{children}</section>; }
function statusTone(value: string) {
  if (["Conectado", "Ativa", "Ativo", "Em andamento", "Configurada", "Liberado"].includes(value)) return "state-positive";
  if (["Pausado", "Aguardando", "Pausa solicitada", "Bloqueio solicitado"].includes(value)) return "state-warning";
  if (["Desconectado", "Offline", "Inativo", "Não configurada", "Bloqueado"].includes(value)) return "state-negative";
  return "";
}
function Row({ Icon, label, value }: { Icon: typeof Clock3; label: string; value: string }) { return <div className="info-row"><Icon size={20} /><span>{label}</span><strong className={statusTone(value)}>{value}</strong></div>; }

function Limits({ device, onSave }: { device: Device; onSave: (weekly: number[]) => Promise<void> }) {
  const [draft, setDraft] = useState([...device.weekly_quota_seconds]); const [day, setDay] = useState(new Date().getDay()); const [busy, setBusy] = useState(false); const value = draft[day];
  return <section className="editor-page"><header><div><h2>Limites</h2><p>Defina o tempo disponível em cada dia da semana.</p></div></header><div className="day-tabs">{dayNames.map((name, index) => <button className={day === index ? "active" : ""} key={name} onClick={() => setDay(index)}><span>{name}</span><strong>{fmt(draft[index])}</strong></button>)}</div><section className="limit-editor"><DurationWheel label={`Limite · ${dayNames[day]}`} value={value} onChange={(nextValue) => setDraft((all) => all.map((item, index) => index === day ? nextValue : item))} /><div className="step-actions"><button type="button" onClick={() => setDraft((all) => all.map((item, index) => index === day ? 0 : item))}>Bloquear o dia</button><button type="button" onClick={() => setDraft((all) => all.map((item, index) => index === day ? 86400 : item))}>Liberar o dia</button></div><button className="primary-button" disabled={busy} onClick={async () => { setBusy(true); try { await onSave(draft); } finally { setBusy(false); } }}><Save size={18} />{busy ? "Salvando…" : "Salvar limites"}</button></section></section>;
}
function Routines({ device, onNew, onEdit, onToggle, onDelete }: { device: Device; onNew: () => void; onEdit: (routine: Routine) => void; onToggle: (routine: Routine) => Promise<void>; onDelete: (routine: Routine) => Promise<void> }) {
  return <section className="editor-page"><header><div><h2>Rotinas</h2><p>Crie ritmos automáticos para estudo, descanso e lazer.</p></div><button className="primary-button" onClick={onNew}><Plus size={18} />Nova rotina</button></header><div className="routine-list">{device.routines.length ? device.routines.map((routine) => <article key={routine.id}><RoutineVisual iconKey={routineIconFor(routine)} /><span className="routine-time">{clock(routine.start_second)}<i />{clock(routine.end_second)}</span><div><h3>{routine.name}</h3><p>{routine.days.map((selected, index) => selected ? dayNames[index] : "").filter(Boolean).join(" · ")}</p></div><button aria-pressed={routine.enabled} className={`switch ${routine.enabled ? "active" : ""}`} onClick={() => void onToggle(routine)}><i /></button><button aria-label={`Editar ${routine.name}`} className="edit-routine" onClick={() => onEdit(routine)}><Pencil size={18} /></button><button aria-label={`Excluir ${routine.name}`} className="delete" onClick={() => void onDelete(routine)}><Trash2 size={18} /></button></article>) : <div className="empty"><CalendarDays size={28} /><h3>Nenhuma rotina criada</h3><p>Comece organizando um período recorrente.</p></div>}</div></section>;
}

function Administration({ device, onSave, onPassword, onIssueToken, onRevokeToken, onDelete }: { device: Device; onSave: (name: string, warning: number, avatarKey: AvatarKey) => Promise<void>; onPassword: (password: string, confirmation: string) => Promise<void>; onIssueToken: () => Promise<string>; onRevokeToken: () => Promise<void>; onDelete: () => Promise<void> }) {
  const [name, setName] = useState(device.name); const [warning, setWarning] = useState(device.warning_minutes); const [avatarKey, setAvatarKey] = useState<AvatarKey>(avatarKeyFor(device)); const [busy, setBusy] = useState(false);
  const [password, setPassword] = useState(""); const [confirmation, setConfirmation] = useState(""); const [passwordBusy, setPasswordBusy] = useState(false); const [passwordTouched, setPasswordTouched] = useState(false);
  const [token, setToken] = useState(""); const [tokenBusy, setTokenBusy] = useState(false); const [credentialAction, setCredentialAction] = useState<"issue" | "revoke" | null>(null); const [copyFeedback, setCopyFeedback] = useState<{ kind: "id" | "token"; status: "success" | "error" } | null>(null); const copyFeedbackTimer = useRef<number | null>(null);
  const [deleteOpen, setDeleteOpen] = useState(false); const [deleteBusy, setDeleteBusy] = useState(false); const [deleteError, setDeleteError] = useState("");
  const passwordError = passwordTouched && !password ? "Informe uma senha." : passwordTouched && password !== confirmation ? "As senhas não coincidem." : "";
  useEffect(() => { setName(device.name); setWarning(device.warning_minutes); setAvatarKey(avatarKeyFor(device)); }, [device.id]);
  useEffect(() => () => { if (copyFeedbackTimer.current !== null) window.clearTimeout(copyFeedbackTimer.current); }, []);
  const copyValue = async (value: string, kind: "id" | "token") => {
    if (copyFeedbackTimer.current !== null) window.clearTimeout(copyFeedbackTimer.current);
    try {
      await copyVisibleValue(value);
      setCopyFeedback({ kind, status: "success" });
      try { if ("vibrate" in navigator) navigator.vibrate(24); } catch { /* confirmação visual permanece disponível */ }
    } catch {
      setCopyFeedback({ kind, status: "error" });
    }
    copyFeedbackTimer.current = window.setTimeout(() => setCopyFeedback(null), 2200);
  };
  const copyButton = (value: string, kind: "id" | "token", label: string) => {
    const status = copyFeedback?.kind === kind ? copyFeedback.status : "idle";
    const text = status === "success" ? "Copiado!" : status === "error" ? "Não copiou" : "Copiar";
    return <button aria-label={status === "success" ? `${label} copiado` : status === "error" ? `Não foi possível copiar ${label.toLowerCase()}` : `Copiar ${label.toLowerCase()}`} className={`copy-button ${status}`} onClick={() => void copyValue(value, kind)} type="button">{status === "success" ? <Check aria-hidden="true" size={17} /> : status === "error" ? <X aria-hidden="true" size={17} /> : <Copy aria-hidden="true" size={17} />}<span aria-live="polite">{text}</span></button>;
  };
  return <section className="editor-page"><header><div><h2>Administração</h2><p>Identidade, acesso e segurança deste computador.</p></div></header><div className="administration-sections">
    <section className="admin-section identity-section"><div className="admin-section-heading"><Settings size={21} /><div><h3>Identidade</h3><p>Nome, avatar e aviso de encerramento.</p></div></div><form className="admin-form" onSubmit={async (event) => { event.preventDefault(); setBusy(true); try { await onSave(name.trim(), warning, avatarKey); } finally { setBusy(false); } }}><label>Nome do computador<input value={name} onChange={(event) => setName(event.target.value)} /></label><AvatarPicker value={avatarKey} onChange={setAvatarKey} /><label>Aviso antes do fim<select value={warning} onChange={(event) => setWarning(Number(event.target.value))}>{[5, 10, 15, 30].map((item) => <option key={item} value={item}>{item} minutos</option>)}</select></label><button className="primary-button" disabled={!name.trim() || busy}><Save size={18} />{busy ? "Salvando…" : "Salvar informações"}</button></form></section>
    <section className="admin-section"><div className="admin-section-heading"><LockKeyhole size={21} /><div><h3>{device.password_set ? "Alterar senha local" : "Configurar senha local"}</h3><p>Autoriza tempo adicional diretamente no computador.</p></div></div><form className="admin-form" noValidate onSubmit={async (event) => { event.preventDefault(); setPasswordTouched(true); if (!password || password !== confirmation) return; setPasswordBusy(true); try { await onPassword(password, confirmation); setPassword(""); setConfirmation(""); setPasswordTouched(false); } finally { setPasswordBusy(false); } }}><label>Nova senha<input aria-invalid={Boolean(passwordError)} autoComplete="new-password" type="password" value={password} onBlur={() => setPasswordTouched(true)} onChange={(event) => setPassword(event.target.value)} /></label><label>Confirmar senha<input aria-invalid={Boolean(passwordError)} autoComplete="new-password" type="password" value={confirmation} onBlur={() => setPasswordTouched(true)} onChange={(event) => setConfirmation(event.target.value)} /></label>{passwordError && <p className="field-error"><X size={15} />{passwordError}</p>}<button className="primary-button" disabled={!password || password !== confirmation || passwordBusy}><Save size={18} />{passwordBusy ? "Salvando…" : "Salvar senha"}</button></form></section>
    <section className="admin-section pairing-section"><div className="admin-section-heading"><KeyRound size={21} /><div><h3>Liberar acesso do agente</h3><p>Use estes dados para conectar esta máquina.</p></div></div><div className="credential-field"><span>Identificador <code>device_id</code></span><div><code>{device.id}</code>{copyButton(device.id, "id", "Identificador")}</div></div>{token ? <div className="credential-field token-reveal"><span>Token — copie agora <code>device_token</code></span><p>Este token será exibido somente desta vez.</p><div><code>{token}</code>{copyButton(token, "token", "Token")}</div></div> : <p className="credential-note">O token atual não pode ser consultado. Gere um novo apenas quando for configurar o agente.</p>}<div className="credential-actions"><button className="primary-button" disabled={tokenBusy} onClick={() => setCredentialAction("issue")}><RefreshCw size={18} />Gerar novo token</button><button className="danger-button" disabled={tokenBusy} onClick={() => setCredentialAction("revoke")}><ShieldOff size={18} />Revogar token</button></div></section>
    <section className="technical admin-section"><div className="admin-section-heading"><MonitorCheck size={21} /><div><h3>Estado técnico</h3><p>Aplicação e sincronização.</p></div></div><Row Icon={ShieldCheck} label="Senha local" value={device.password_set ? "Configurada" : "Não configurada"} /><Row Icon={RefreshCw} label="Última sincronização" value={lastSeen(device.last_seen_at)} /></section>
    <section className="admin-section device-danger-zone"><div className="admin-section-heading"><Trash2 size={21} /><div><h3>Excluir computador</h3><p>Remove este computador e todas as suas configurações.</p></div></div><button className="danger-button" onClick={() => { setDeleteError(""); setDeleteOpen(true); }}><Trash2 size={18} />Excluir computador</button></section>
  </div>{credentialAction && <Modal title={credentialAction === "issue" ? "Gerar novo token?" : "Revogar token?"} description={credentialAction === "issue" ? "Se existir um token anterior, ele deixará de funcionar imediatamente." : "O agente perderá o acesso até que um novo token seja configurado."} onClose={() => setCredentialAction(null)}><div className="modal-actions"><button onClick={() => setCredentialAction(null)}>Cancelar</button><button className={credentialAction === "issue" ? "primary" : "danger-confirm"} disabled={tokenBusy} onClick={async () => { setTokenBusy(true); try { if (credentialAction === "issue") setToken(await onIssueToken()); else { await onRevokeToken(); setToken(""); } setCredentialAction(null); } finally { setTokenBusy(false); } }}>{credentialAction === "issue" ? "Gerar token" : "Revogar acesso"}</button></div></Modal>}{deleteOpen && <Modal title="Excluir computador?" description={`“${device.name}” será removido permanentemente. O agente perderá o acesso e precisará ser cadastrado novamente.`} onClose={() => !deleteBusy && setDeleteOpen(false)}>{deleteError && <div className="routine-conflict-alert" role="alert"><AlertTriangle aria-hidden="true" size={20} /><div><strong>Não foi possível excluir</strong><span>{deleteError}</span></div></div>}<div className="modal-actions"><button disabled={deleteBusy} onClick={() => setDeleteOpen(false)}>Cancelar</button><button className="danger-confirm" disabled={deleteBusy} onClick={async () => { setDeleteBusy(true); setDeleteError(""); try { await onDelete(); setDeleteOpen(false); } catch (error) { setDeleteError(error instanceof Error ? error.message : "Tente novamente."); } finally { setDeleteBusy(false); } }}><Trash2 size={18} />{deleteBusy ? "Excluindo…" : "Excluir"}</button></div></Modal>}</section>;
}

function mockToken() { return Array.from(crypto.getRandomValues(new Uint8Array(43)), (byte) => "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"[byte % 64]).join(""); }

function BonusModal({ onClose, onSubmit }: { onClose: () => void; onSubmit: (minutes: number) => Promise<void> }) { const [value, setValue] = useState(30); const [busy, setBusy] = useState(false); const [error, setError] = useState(""); return <Modal title="Mais tempo" description="O tempo extra vale somente para hoje." onClose={onClose}><div className="preset-grid">{[15, 30, 45, 60].map((item) => <button className={value === item ? "active" : ""} key={item} onClick={() => { setValue(item); setError(""); }}><strong>{item}</strong><span>min</span></button>)}</div>{error && <div className="routine-conflict-alert" role="alert"><AlertTriangle aria-hidden="true" size={20} /><div><strong>Tempo não adicionado</strong><span>{error}</span></div></div>}<div className="modal-actions"><button onClick={onClose}>Cancelar</button><button className="primary" disabled={busy} onClick={async () => { setBusy(true); setError(""); try { await onSubmit(value); } catch (submitError) { setError(submitError instanceof Error ? submitError.message : "Não foi possível adicionar tempo."); } finally { setBusy(false); } }}><Plus size={18} />Adicionar {value}min</button></div></Modal>; }
function DeviceModal({ onClose, onSubmit }: { onClose: () => void; onSubmit: (name: string, avatarKey: AvatarKey) => Promise<void> }) { const [name, setName] = useState(""); const [avatarKey, setAvatarKey] = useState<AvatarKey>("cat"); const [busy, setBusy] = useState(false); return <Modal title="Novo computador" description="Dê um nome e uma identidade antes de parear o agente." onClose={onClose}><form className="modal-form" onSubmit={async (event) => { event.preventDefault(); setBusy(true); try { await onSubmit(name.trim(), avatarKey); } finally { setBusy(false); } }}><label>Nome<input autoFocus placeholder="Ex.: Notebook da Ana" value={name} onChange={(event) => setName(event.target.value)} /></label><AvatarPicker value={avatarKey} onChange={setAvatarKey} /><div className="modal-actions"><button type="button" onClick={onClose}>Cancelar</button><button className="primary" disabled={!name.trim() || busy}><Plus size={18} />Adicionar</button></div></form></Modal>; }
function RoutineModal({ initial, routines, onClose, onSubmit }: { initial?: Routine; routines: Routine[]; onClose: () => void; onSubmit: (routine: Omit<Routine, "id">) => Promise<void> }) { const [name, setName] = useState(initial?.name ?? ""); const [iconKey, setIconKey] = useState<RoutineIconKey>(initial ? routineIconFor(initial) : "study"); const [iconTouched, setIconTouched] = useState(Boolean(initial)); const [start, setStart] = useState(initial ? clock(initial.start_second) : "18:30"); const [end, setEnd] = useState(initial ? clock(initial.end_second) : "20:00"); const [days, setDays] = useState(initial ? initial.days.map((selected, index) => selected ? index : -1).filter((index) => index >= 0) : [1,2,3,4,5]); const [busy, setBusy] = useState(false); const [serverError, setServerError] = useState(""); const selectedIcon = iconTouched ? iconKey : inferRoutineIcon(name); const draft = { name: name.trim(), icon_key: selectedIcon, start_second: seconds(start), end_second: seconds(end), days: dayNames.map((_, index) => days.includes(index)) as Routine["days"], enabled: initial?.enabled ?? true }; const conflict = conflictingRoutine(draft, routines); const error = conflict ? `Este intervalo já está ocupado pela rotina “${conflict.name}”.` : serverError; return <Modal title={initial ? "Editar rotina" : "Nova rotina"} description="Escolha uma identidade, horários e dias." onClose={onClose}><form className="modal-form" onSubmit={async (event) => { event.preventDefault(); if (conflict) return; setBusy(true); setServerError(""); try { await onSubmit(draft); } catch (submitError) { setServerError(submitError instanceof Error ? submitError.message : `Não foi possível ${initial ? "alterar" : "criar"} a rotina.`); } finally { setBusy(false); } }}><label>Nome<input autoFocus placeholder="Ex.: Tempo de estudo" value={name} onChange={(event) => setName(event.target.value)} /></label><RoutineIconPicker value={selectedIcon} onChange={(value) => { setIconKey(value); setIconTouched(true); }} /><TimeRangePicker start={start} end={end} onChange={(nextStart, nextEnd) => { setStart(nextStart); setEnd(nextEnd); setServerError(""); }} /><fieldset className="weekday-picker"><legend>Dias</legend>{dayNames.map((day, index) => { const active = days.includes(index); return <button aria-pressed={active} className={active ? "active" : ""} key={day} onClick={() => { setDays((all) => all.includes(index) ? all.filter((item) => item !== index) : [...all, index]); setServerError(""); }} type="button">{active && <Check size={14} />}{day}</button>; })}</fieldset>{error && <div className="routine-conflict-alert" role="alert"><AlertTriangle aria-hidden="true" size={20} /><div><strong>{conflict ? "Horário indisponível" : `Não foi possível ${initial ? "alterar" : "criar"} a rotina`}</strong><span>{error}</span></div></div>}<div className="modal-actions"><button type="button" onClick={onClose}>Cancelar</button><button className="primary" disabled={!name.trim() || !days.length || Boolean(conflict) || busy}><CalendarDays size={18} />{initial ? "Salvar rotina" : "Criar rotina"}</button></div></form></Modal>; }
