import { Activity, CalendarDays, Plus, RefreshCw, Settings, SlidersHorizontal, SquareTerminal } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { api, remoteMode } from "./api";
import { Brand, Toast } from "./components";
import { CommunicationPage } from "./communication/CommunicationPage";
import { AdministrationPage } from "./features/administration/AdministrationPage";
import { LoginPage } from "./features/auth/LoginPage";
import { lastSeen } from "./features/common/format";
import { DeviceModal } from "./features/devices/DeviceModal";
import { DeviceRail, MobileDeviceHeader } from "./features/devices/DeviceNavigation";
import { avatarKeyFor, DeviceState, deviceIsBlockedForAction } from "./features/devices/devicePresentation";
import { LimitsPage } from "./features/limits/LimitsPage";
import { BonusModal } from "./features/now/BonusModal";
import { NowPage } from "./features/now/NowPage";
import { RoutineModal } from "./features/routines/RoutineModal";
import { RoutinesPage } from "./features/routines/RoutinesPage";
import { withoutId } from "./features/routines/routineSchedule";
import { useDeviceStream } from "./hooks/useDeviceStream";
import { useNotifications } from "./hooks/useNotifications";
import { mockDevices } from "./mock";
import type { Device, Routine, View } from "./types";
import { DeviceAvatar } from "./visuals";

const nav: { id: View; label: string; short?: string; Icon: typeof Activity }[] = [
  { id: "now", label: "Agora", Icon: Activity },
  { id: "limits", label: "Limites", Icon: SlidersHorizontal },
  { id: "routines", label: "Rotinas", Icon: CalendarDays },
  { id: "administration", label: "Administração", short: "Admin.", Icon: Settings },
  { id: "communication", label: "Atividade", short: "Ativ.", Icon: SquareTerminal },
];

function localID() {
  return typeof globalThis.crypto?.randomUUID === "function"
    ? globalThis.crypto.randomUUID()
    : `local-${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`;
}

function mockToken() {
  return Array.from(crypto.getRandomValues(new Uint8Array(43)), (byte) => "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"[byte % 64]).join("");
}

export function App() {
  const [devices, setDevices] = useState<Device[]>(remoteMode ? [] : mockDevices);
  const [selectedId, setSelectedId] = useState(mockDevices[0].id);
  const [view, setView] = useState<View>("now");
  const [authenticated, setAuthenticated] = useState(!remoteMode);
  const [checking, setChecking] = useState(remoteMode);
  const [loading, setLoading] = useState(remoteMode);
  const [modal, setModal] = useState<"bonus" | "device" | "routine" | null>(null);
  const [editingRoutine, setEditingRoutine] = useState<Routine | null>(null);
  const pendingOperations = useRef(new Set<string>());
  const selected = useMemo(() => devices.find((item) => item.id === selectedId) ?? devices[0], [devices, selectedId]);
  const { message, notify, setMessage } = useNotifications();
  const { streamState, streamGeneration, activityUpdate, communicationUpdate } = useDeviceStream({
    deviceId: selected?.id,
    enabled: remoteMode,
    setDevices,
    pendingOperations,
    notify,
  });

  const load = async () => {
    setLoading(true);
    try {
      const list = await api.devices();
      setDevices(list);
      if (list.length) setSelectedId((id) => list.some((device) => device.id === id) ? id : list[0].id);
    } catch (error) {
      notify(error instanceof Error ? error.message : "Falha ao carregar dados.");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (!remoteMode) return;
    api.session()
      .then((session) => {
        setAuthenticated(session.authenticated);
        if (session.authenticated) void load();
      })
      .catch(() => setAuthenticated(false))
      .finally(() => setChecking(false));
  }, []);

  const patchDevice = (change: (device: Device) => Device) => {
    setDevices((all) => all.map((item) => item.id === selected.id ? change(item) : item));
  };
  const command = async (name: string, local: (device: Device) => Device, success: string) => {
    try {
      if (remoteMode) {
        const confirmation = await api.command(selected.id, name);
        pendingOperations.current.add(confirmation.operation_id);
      }
      patchDevice(local);
      notify(success);
      if (remoteMode) await load();
    } catch (error) {
      notify(error instanceof Error ? error.message : "Comando não enviado.");
    }
  };
  const logout = async () => {
    if (remoteMode) await api.logout();
    setAuthenticated(false);
  };

  if (checking) return <div className="center-state"><Brand /><span className="loader" />Verificando sessão…</div>;
  if (!authenticated) return <LoginPage onLogin={async (login, password) => { const session = await api.login(login, password); setAuthenticated(session.authenticated); if (session.authenticated) await load(); return session.authenticated; }} />;
  if (loading && !selected) return <div className="center-state"><Brand /><span className="loader" />Organizando seus computadores…</div>;
  if (!selected) return <div className="center-state"><Brand /><h1>Nenhum computador</h1><button className="primary-button" onClick={() => setModal("device")}><Plus size={18} />Adicionar computador</button>{modal === "device" && <DeviceModal onClose={() => setModal(null)} onSubmit={async (name, avatarKey) => { if (remoteMode) await api.createDevice(name, avatarKey); else setDevices([{ ...mockDevices[1], id: localID(), name, avatar_key: avatarKey }]); setModal(null); if (remoteMode) await load(); }} />}{message && <Toast>{message}</Toast>}</div>;

  return <div className="app-shell">
    <a className="skip" href="#workspace">Pular para o conteúdo</a>
    <DeviceRail devices={devices} selected={selected} onSelect={(deviceId) => { setSelectedId(deviceId); setView("now"); }} onAdd={() => setModal("device")} onLogout={() => void logout()} />
    <aside className="main-nav"><div className="nav-title">Ações</div><nav>{nav.map(({ id, label, Icon }) => <button className={view === id ? "active" : ""} key={id} onClick={() => setView(id)}><Icon size={20} />{label}</button>)}</nav></aside>
    <main id="workspace">
      <MobileDeviceHeader devices={devices} selected={selected} onSelect={setSelectedId} onLogout={() => void logout()} />
      <header className="workspace-header"><div><DeviceAvatar avatarKey={avatarKeyFor(selected)} className="workspace-avatar" name={selected.name} /><span><h1>{selected.name}</h1><DeviceState device={selected} /></span></div><button onClick={() => void load()}><RefreshCw size={17} />{lastSeen(selected.last_seen_at)}</button></header>
      <div className={`workspace-body ${view === "communication" ? "communication-workspace" : ""}`}>
        {view === "now" && <NowPage device={selected} onBonus={() => setModal("bonus")} onPause={() => void command(selected.monitoring_paused ? "resume_monitoring" : "pause_monitoring", (device) => ({ ...device, monitoring_paused: !device.monitoring_paused, manual_block: false, actual_state: remoteMode ? device.actual_state : "unblocked", control_status: device.monitoring_paused ? remoteMode ? "resume_requested" : "active" : remoteMode ? "pause_requested" : "paused" }), selected.monitoring_paused ? "Retomada solicitada." : "Pausa solicitada.")} onBlock={() => { const blockedForAction = deviceIsBlockedForAction(selected); void command(blockedForAction ? "clear_manual_block" : "block_now", (device) => ({ ...device, monitoring_paused: false, manual_block: !blockedForAction, actual_state: blockedForAction ? remoteMode ? device.actual_state : "unblocked" : remoteMode ? device.actual_state : "blocked", control_status: blockedForAction ? remoteMode ? "unblock_requested" : "active" : remoteMode ? "block_requested" : "blocked" }), blockedForAction ? "Desbloqueio solicitado." : "Bloqueio solicitado."); }} />}
        {view === "limits" && <LimitsPage device={selected} onSave={async (weekly) => { if (remoteMode) await api.policy(selected.id, weekly, selected.warning_minutes); patchDevice((device) => ({ ...device, weekly_quota_seconds: weekly })); notify("Limites salvos."); if (remoteMode) await load(); }} />}
        {view === "routines" && <RoutinesPage device={selected} onNew={() => { setEditingRoutine(null); setModal("routine"); }} onEdit={(routine) => { setEditingRoutine(routine); setModal("routine"); }} onToggle={async (routine) => { const next = { ...routine, enabled: !routine.enabled }; if (remoteMode) await api.routine(selected.id, withoutId(next), routine.id); patchDevice((device) => ({ ...device, routines: device.routines.map((item) => item.id === routine.id ? next : item) })); notify(next.enabled ? "Rotina ativada." : "Rotina pausada."); }} onDelete={async (routine) => { if (remoteMode) await api.deleteRoutine(selected.id, routine.id); patchDevice((device) => ({ ...device, routines: device.routines.filter((item) => item.id !== routine.id) })); notify("Rotina removida."); }} />}
        {view === "administration" && <AdministrationPage device={selected} onSave={async (name, warning, avatarKey) => { if (remoteMode) { await api.rename(selected.id, name, avatarKey); await api.policy(selected.id, selected.weekly_quota_seconds, warning); await load(); } patchDevice((device) => ({ ...device, name, warning_minutes: warning, avatar_key: avatarKey })); notify("Informações salvas."); }} onPassword={async (password, confirmation) => { if (remoteMode) await api.updatePassword(selected.id, password, confirmation); patchDevice((device) => ({ ...device, password_set: true })); notify("Senha local atualizada."); if (remoteMode) await load(); }} onIssueToken={async () => { const result = remoteMode ? await api.issueToken(selected.id) : { device_id: selected.id, device_token: mockToken() }; notify("Novo token gerado."); return result.device_token; }} onRevokeToken={async () => { if (remoteMode) await api.revokeToken(selected.id); notify("Token revogado."); }} onDelete={async () => { if (remoteMode) await api.deleteDevice(selected.id); const remaining = devices.filter((device) => device.id !== selected.id); setDevices(remaining); setSelectedId(remaining[0]?.id ?? ""); setView("now"); notify("Computador excluído."); }} />}
        {view === "communication" && <CommunicationPage activityUpdate={activityUpdate} communicationUpdate={communicationUpdate} deviceId={selected.id} deviceName={selected.name} streamGeneration={streamGeneration} streamState={streamState} />}
      </div>
    </main>
    <nav className="bottom-nav">{nav.map(({ id, label, short, Icon }) => <button className={view === id ? "active" : ""} key={id} onClick={() => setView(id)}><Icon size={21} /><span>{short ?? label}</span></button>)}</nav>
    {modal === "bonus" && <BonusModal onClose={() => setModal(null)} onSubmit={async (minutes) => { if (!remoteMode) { patchDevice((device) => ({ ...device, bonus_seconds: device.bonus_seconds + minutes * 60, remaining_seconds: device.remaining_seconds + minutes * 60 })); setModal(null); notify(`${minutes} minutos adicionados.`); return; } const confirmation = await api.bonus(selected.id, minutes); pendingOperations.current.add(confirmation.operation_id); setModal(null); setMessage("Pedido guardado pelo servidor. Aguardando o computador confirmar."); }} />}
    {modal === "device" && <DeviceModal onClose={() => setModal(null)} onSubmit={async (name, avatarKey) => { if (remoteMode) { const created = await api.createDevice(name, avatarKey); setSelectedId(created.id); await load(); } else { const device = { ...mockDevices[1], id: localID(), name, avatar_key: avatarKey }; setDevices((all) => [...all, device]); setSelectedId(device.id); } setModal(null); notify("Computador adicionado."); }} />}
    {modal === "routine" && <RoutineModal initial={editingRoutine ?? undefined} routines={selected.routines.filter((routine) => routine.id !== editingRoutine?.id)} onClose={() => { setModal(null); setEditingRoutine(null); }} onSubmit={async (draft) => { const routineId = editingRoutine?.id; const saved = remoteMode ? await api.routine(selected.id, draft, routineId) : { id: routineId ?? localID() }; const savedId = routineId ?? saved.id; if (remoteMode) await load(); patchDevice((device) => ({ ...device, routines: device.routines.some((routine) => routine.id === savedId) ? device.routines.map((routine) => routine.id === savedId ? { ...draft, id: savedId } : routine) : [...device.routines, { ...draft, id: savedId || localID() }] })); setModal(null); setEditingRoutine(null); notify(routineId ? "Rotina atualizada." : "Rotina criada."); }} />}
    {message && <Toast>{message}</Toast>}
  </div>;
}
