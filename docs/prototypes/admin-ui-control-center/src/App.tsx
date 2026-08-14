import { useMemo, useState, type FormEvent, type ReactNode } from "react";
import {
  Activity, Bell, BookOpen, CalendarDays, Check, ChevronDown, Clock3, Computer,
  Gauge, Home, Laptop, ListFilter, LockKeyhole, LogOut, Menu, Minus, Moon,
  MoreHorizontal, Pause, Pencil, Play, Plus, RefreshCw, Save, Settings,
  ShieldCheck, SlidersHorizontal, Sparkles, Trash2, UserRound, X,
} from "lucide-react";
import { initialDevices } from "./data";
import type { Device, EventItem, Routine, View } from "./types";

const dayLabels = ["Dom", "Seg", "Ter", "Qua", "Qui", "Sex", "Sáb"];
const mainDays = [1, 2, 3, 4, 5, 6, 0];

function minutes(value: number) {
  if (value <= 0) return "—";
  const h = Math.floor(value / 60);
  const m = value % 60;
  return `${h ? `${h}h` : ""}${m ? ` ${m}min` : ""}`.trim();
}

function Mark() {
  return <span className="mark" aria-hidden="true"><span /><i /></span>;
}

function IconButton({ label, children, onClick }: { label: string; children: ReactNode; onClick?: () => void }) {
  return <button className="icon-button" aria-label={label} title={label} onClick={onClick}>{children}</button>;
}

function NavItem({ active, icon, label, onClick }: { active: boolean; icon: ReactNode; label: string; onClick: () => void }) {
  return <button className={`nav-item ${active ? "active" : ""}`} onClick={onClick}>{icon}<span>{label}</span></button>;
}

function Modal({ title, description, children, onClose }: { title: string; description?: string; children: ReactNode; onClose: () => void }) {
  return <div className="modal-backdrop" role="presentation" onMouseDown={(e) => e.target === e.currentTarget && onClose()}>
    <section className="modal" role="dialog" aria-modal="true" aria-labelledby="modal-title">
      <header><div><h2 id="modal-title">{title}</h2>{description && <p>{description}</p>}</div><IconButton label="Fechar" onClick={onClose}><X size={20} /></IconButton></header>
      {children}
    </section>
  </div>;
}

function TimeRing({ device }: { device: Device }) {
  const remaining = Math.max(0, device.limitMinutes + device.bonusMinutes - device.usedMinutes);
  const pct = Math.min(100, Math.max(0, (remaining / Math.max(device.limitMinutes + device.bonusMinutes, 1)) * 100));
  return <div className="time-ring" style={{ "--progress": `${pct * 3.6}deg` } as React.CSSProperties}>
    <div><strong>{minutes(remaining)}</strong><span>{device.blocked ? "acesso bloqueado" : device.paused ? "contagem pausada" : "restantes hoje"}</span></div>
  </div>;
}

function DeviceRail({ devices, selected, setSelected, onAdd }: { devices: Device[]; selected: string; setSelected: (id: string) => void; onAdd: () => void }) {
  return <aside className="device-rail">
    <div className="rail-heading"><h2>Computadores</h2><IconButton label="Adicionar computador" onClick={onAdd}><Plus size={20} /></IconButton></div>
    <div className="device-list">{devices.map((device) =>
      <button key={device.id} className={`device-item ${selected === device.id ? "active" : ""}`} onClick={() => setSelected(device.id)}>
        <span className="device-icon"><Laptop size={25} /></span>
        <span><strong>{device.name}</strong><small className={device.online ? "online" : ""}><i />{device.online ? "Online" : "Offline"}</small></span>
      </button>)}
    </div>
    <button className="add-device" onClick={onAdd}><Plus size={18} />Adicionar computador</button>
  </aside>;
}

function StatusActions({ device, onBonus, onPause, onBlock }: { device: Device; onBonus: () => void; onPause: () => void; onBlock: () => void }) {
  return <div className="status-card">
    <TimeRing device={device} />
    <div className="usage-facts">
      <div><span className="fact-icon"><Clock3 size={20} /></span><p><strong>{minutes(device.usedMinutes)}</strong><small>usados</small></p></div>
      <div><span className="fact-icon"><Gauge size={20} /></span><p><small>Limite</small><strong>{minutes(device.limitMinutes + device.bonusMinutes)}</strong></p></div>
    </div>
    <div className="quick-actions">
      <button className="button primary" onClick={onBonus}><Plus size={19} />Adicionar tempo</button>
      <button className="button secondary" onClick={onPause}>{device.paused ? <Play size={18} /> : <Pause size={18} />}{device.paused ? "Retomar contagem" : "Pausar contagem"}</button>
      <button className={`button ${device.blocked ? "secondary" : "danger"}`} onClick={onBlock}><LockKeyhole size={18} />{device.blocked ? "Desbloquear" : "Bloquear agora"}</button>
    </div>
  </div>;
}

function LimitsPreview({ device, onEdit }: { device: Device; onEdit: () => void }) {
  return <section className="panel limits-preview">
    <div className="panel-heading"><div><h3>Limites da semana</h3><p>Tempo disponível a cada dia</p></div><button className="text-button" onClick={onEdit}><Pencil size={15} />Editar limites</button></div>
    <div className="week-preview">{mainDays.map((day) => <div key={day}><span>{dayLabels[day]}</span><strong>{minutes(device.limits[day])}</strong><i style={{ width: `${Math.max(12, device.limits[day] / 2.4)}%` }} /></div>)}</div>
    <p className="hint"><Clock3 size={15} />Os limites diários se renovam à meia-noite.</p>
  </section>;
}

function RoutineGlyph({ kind }: { kind: Routine["kind"] }) {
  return <span className={`routine-glyph ${kind}`}>{kind === "study" ? <BookOpen size={21} /> : kind === "sleep" ? <Moon size={21} /> : <Sparkles size={21} />}</span>;
}

function RoutinesPreview({ routines, onOpen, onNew }: { routines: Routine[]; onOpen: () => void; onNew: () => void }) {
  return <section className="panel routines-preview">
    <div className="panel-heading"><div><h3>Rotinas de hoje</h3><p>Próximos períodos programados</p></div><button className="text-button" onClick={onNew}><Plus size={16} />Nova rotina</button></div>
    <div className="routine-timeline">{routines.filter((r) => r.enabled).slice(0, 2).map((routine, i) => <button key={routine.id} onClick={onOpen}>
      <RoutineGlyph kind={routine.kind} /><span><strong>{routine.name}</strong><small>{routine.start} – {routine.end}</small></span><em>{i === 0 ? "Em andamento" : "Agendada"}</em>
    </button>)}</div>
  </section>;
}

function ActivityList({ events, compact = false }: { events: EventItem[]; compact?: boolean }) {
  const visible = compact ? events.slice(0, 3) : events;
  return <div className="activity-list">{visible.length ? visible.map((event) => <div key={event.id}>
    <span className={`event-dot ${event.tone}`}><Activity size={17} /></span><p><strong>{event.title}</strong><small>{event.detail}</small></p><time>{event.time}</time>
  </div>) : <div className="empty"><Activity size={28} /><strong>Nenhuma atividade recente</strong><span>As alterações e sincronizações aparecerão aqui.</span></div>}</div>;
}

function HomeView({ device, setView, onBonus, onPause, onBlock, onRoutine }: { device: Device; setView: (v: View) => void; onBonus: () => void; onPause: () => void; onBlock: () => void; onRoutine: () => void }) {
  return <>
    <div className="device-title"><span className="device-hero-icon"><Laptop size={32} /></span><div><h2>{device.name}</h2><p><i className={device.online ? "online" : ""} />{device.online ? `Online · ${device.inUse ? "em uso" : "sem sessão ativa"}` : `Offline · visto ${device.lastSeen}`}</p></div><button className="details-button" onClick={() => setView("settings")}>Ver detalhes<ChevronDown size={17} /></button></div>
    <StatusActions device={device} onBonus={onBonus} onPause={onPause} onBlock={onBlock} />
    <div className="dashboard-grid"><LimitsPreview device={device} onEdit={() => setView("limits")} /><RoutinesPreview routines={device.routines} onOpen={() => setView("routines")} onNew={onRoutine} /></div>
    <section className="panel activity-preview"><div className="panel-heading"><div><h3>Atividade recente</h3><p>Comandos e alterações deste computador</p></div><button className="text-button" onClick={() => setView("activity")}>Ver tudo</button></div><ActivityList events={device.events} compact /></section>
  </>;
}

function LimitsView({ device, onSave }: { device: Device; onSave: (limits: number[]) => void }) {
  const [draft, setDraft] = useState([...device.limits]);
  const [selected, setSelected] = useState(new Date().getDay());
  return <section className="content-page">
    <div className="page-heading"><div><h2>Limites da semana</h2><p>Defina quanto tempo {device.name} pode ser usado em cada dia.</p></div><button className="button primary compact" onClick={() => onSave(draft)}><Save size={17} />Salvar alterações</button></div>
    <div className="limits-editor panel"><div className="day-tabs">{mainDays.map((day) => <button className={selected === day ? "active" : ""} key={day} onClick={() => setSelected(day)}><span>{dayLabels[day]}</span><strong>{minutes(draft[day])}</strong></button>)}</div>
      <div className="dial-area"><div className="dial" style={{ "--dial": `${Math.min(360, draft[selected] / 4)}deg` } as React.CSSProperties}><span><strong>{minutes(draft[selected])}</strong><small>neste dia</small></span></div>
        <div className="dial-controls"><h3>{dayLabels[selected]}-feira</h3><p>Ajuste em intervalos de 15 minutos.</p><div><button onClick={() => setDraft((v) => v.map((n, i) => i === selected ? Math.max(0, n - 15) : n))}><Minus size={20} /></button><strong>{minutes(draft[selected])}</strong><button onClick={() => setDraft((v) => v.map((n, i) => i === selected ? Math.min(1440, n + 15) : n))}><Plus size={20} /></button></div><span className="quick-set"><button onClick={() => setDraft((v) => v.map((n, i) => i === selected ? 0 : n))}>Bloquear o dia</button><button onClick={() => setDraft((v) => v.map((n, i) => i === selected ? 1440 : n))}>Liberar o dia</button></span></div>
      </div>
    </div>
  </section>;
}

function RoutinesView({ device, onToggle, onNew }: { device: Device; onToggle: (id: string) => void; onNew: () => void }) {
  return <section className="content-page"><div className="page-heading"><div><h2>Rotinas</h2><p>Organize horários de estudo, lazer e descanso.</p></div><button className="button primary compact" onClick={onNew}><Plus size={18} />Nova rotina</button></div>
    <div className="routine-list">{device.routines.map((routine) => <article className="routine-row panel" key={routine.id}><RoutineGlyph kind={routine.kind} /><div className="routine-main"><h3>{routine.name}</h3><p>{routine.start} – {routine.end}</p><span>{routine.days.map((day) => dayLabels[day]).join(" · ")}</span></div><label className="switch"><input type="checkbox" checked={routine.enabled} onChange={() => onToggle(routine.id)} /><span /></label><IconButton label={`Editar ${routine.name}`}><Pencil size={18} /></IconButton><IconButton label={`Mais opções para ${routine.name}`}><MoreHorizontal size={19} /></IconButton></article>)}</div>
  </section>;
}

function ActivityView({ device }: { device: Device }) {
  return <section className="content-page"><div className="page-heading"><div><h2>Atividade</h2><p>Histórico de comandos, rotinas e sincronizações.</p></div><button className="button secondary compact"><ListFilter size={17} />Filtrar</button></div><section className="panel activity-full"><div className="date-label">Hoje</div><ActivityList events={device.events} /><div className="date-label">Ontem</div><ActivityList events={[{ id: "old", title: "Política sincronizada", detail: "Limites recebidos pelo agente", time: "22:18", tone: "green" }]} /></section></section>;
}

function SettingsView({ device, onRename }: { device: Device; onRename: (name: string) => void }) {
  const [name, setName] = useState(device.name);
  return <section className="content-page"><div className="page-heading"><div><h2>Configurações</h2><p>Identidade, segurança e pareamento deste computador.</p></div></div>
    <div className="settings-layout"><section className="panel settings-form"><h3>Informações do computador</h3><label>Nome do computador<input value={name} onChange={(e) => setName(e.target.value)} /></label><label>Aviso antes do fim<select defaultValue={device.warningMinutes}><option value={5}>5 minutos</option><option value={10}>10 minutos</option><option value={15}>15 minutos</option><option value={30}>30 minutos</option></select></label><button className="button primary compact" onClick={() => onRename(name)}><Save size={17} />Salvar informações</button></section>
      <section className="panel security-list"><h3>Segurança e agente</h3><button><span><ShieldCheck size={21} /></span><p><strong>Credencial de pareamento</strong><small>Ativa · gerar nova ou revogar</small></p><ChevronDown size={17} /></button><button><span><LockKeyhole size={21} /></span><p><strong>Senha local</strong><small>Configurada · alterar senha</small></p><ChevronDown size={17} /></button><button><span><RefreshCw size={21} /></span><p><strong>Sincronização</strong><small>Revisão aplicada · visto {device.lastSeen}</small></p><ChevronDown size={17} /></button><button className="destructive"><span><Trash2 size={21} /></span><p><strong>Remover computador</strong><small>Revoga o acesso e apaga suas regras</small></p><ChevronDown size={17} /></button></section></div>
  </section>;
}

export function App() {
  const [devices, setDevices] = useState(initialDevices);
  const [selectedId, setSelectedId] = useState(initialDevices[0].id);
  const [view, setView] = useState<View>("home");
  const [modal, setModal] = useState<"bonus" | "device" | "routine" | null>(null);
  const [toast, setToast] = useState("");
  const [mobileDevices, setMobileDevices] = useState(false);
  const device = useMemo(() => devices.find((item) => item.id === selectedId) ?? devices[0], [devices, selectedId]);

  function updateDevice(change: (device: Device) => Device, message?: string) {
    setDevices((all) => all.map((item) => item.id === device.id ? change(item) : item));
    if (message) { setToast(message); window.setTimeout(() => setToast(""), 2800); }
  }
  function addEvent(item: Device, event: Omit<EventItem, "id" | "time">) {
    return { ...item, events: [{ ...event, id: crypto.randomUUID(), time: "agora" }, ...item.events] };
  }
  function togglePause() { updateDevice((d) => addEvent({ ...d, paused: !d.paused }, { title: d.paused ? "Contagem retomada" : "Contagem pausada", detail: "Comando enviado por Sérgio", tone: "orange" }), device.paused ? "Contagem retomada" : "Contagem pausada"); }
  function toggleBlock() { updateDevice((d) => addEvent({ ...d, blocked: !d.blocked }, { title: d.blocked ? "Computador desbloqueado" : "Computador bloqueado", detail: "Comando enviado por Sérgio", tone: "orange" }), device.blocked ? "Computador desbloqueado" : "Bloqueio solicitado"); }
  function submitBonus(value: number) { updateDevice((d) => addEvent({ ...d, bonusMinutes: d.bonusMinutes + value }, { title: "Tempo adicionado", detail: `+${value}min por Sérgio`, tone: "blue" }), `${value} minutos adicionados`); setModal(null); }
  function addComputer(name: string) { const id = crypto.randomUUID(); setDevices((all) => [...all, { ...initialDevices[2], id, name, events: [], routines: [] }]); setSelectedId(id); setModal(null); setToast("Computador adicionado"); }
  function addRoutine(name: string, start: string, end: string) { updateDevice((d) => ({ ...d, routines: [...d.routines, { id: crypto.randomUUID(), name, start, end, days: [1,2,3,4,5], enabled: true, kind: "study" }] }), "Rotina criada"); setModal(null); }

  const nav: { id: View; label: string; icon: ReactNode }[] = [
    { id: "home", label: "Visão geral", icon: <Home size={20} /> }, { id: "limits", label: "Limites", icon: <Gauge size={20} /> },
    { id: "routines", label: "Rotinas", icon: <CalendarDays size={20} /> }, { id: "activity", label: "Atividade", icon: <Activity size={20} /> },
  ];

  return <div className="app-shell">
    <a className="skip-link" href="#content">Pular para o conteúdo</a>
    <aside className="sidebar"><div className="brand"><Mark /><span><strong>compasso</strong><small>tempo em família</small></span></div><nav>{nav.map((item) => <NavItem key={item.id} active={view === item.id} icon={item.icon} label={item.label} onClick={() => setView(item.id)} />)}</nav><div className="sidebar-bottom"><NavItem active={view === "settings"} icon={<Settings size={20} />} label="Configurações" onClick={() => setView("settings")} /><button className="profile"><span>SS</span><p><strong>Sérgio</strong><small>Administrador</small></p><ChevronDown size={16} /></button></div></aside>
    <DeviceRail devices={devices} selected={selectedId} setSelected={(id) => { setSelectedId(id); setView("home"); }} onAdd={() => setModal("device")} />
    <main id="content"><header className="topbar"><button className="mobile-brand" onClick={() => setView("home")}><Mark /><strong>compasso</strong></button><div className="welcome"><h1>Olá, Sérgio</h1><p>Tudo em ordem por aqui.</p></div><div className="sync"><Check size={16} />Sincronizado agora</div><IconButton label="Atualizar dados" onClick={() => setToast("Dados atualizados agora")}><RefreshCw size={19} /></IconButton><IconButton label="Notificações"><Bell size={19} /></IconButton></header>
      <button className="mobile-device" onClick={() => setMobileDevices(!mobileDevices)}><span className="device-icon"><Laptop size={22} /></span><span><strong>{device.name}</strong><small className={device.online ? "online" : ""}>{device.online ? "Online · em uso" : "Offline"}</small></span><ChevronDown size={20} /></button>
      {mobileDevices && <div className="mobile-device-menu">{devices.map((d) => <button key={d.id} onClick={() => { setSelectedId(d.id); setMobileDevices(false); }}><Computer size={18} />{d.name}<i className={d.online ? "online" : ""} /></button>)}</div>}
      <div className="workspace">
        {view === "home" && <HomeView device={device} setView={setView} onBonus={() => setModal("bonus")} onPause={togglePause} onBlock={toggleBlock} onRoutine={() => setModal("routine")} />}
        {view === "limits" && <LimitsView key={device.id} device={device} onSave={(limits) => { updateDevice((d) => ({ ...d, limits }), "Limites salvos"); }} />}
        {view === "routines" && <RoutinesView device={device} onNew={() => setModal("routine")} onToggle={(id) => updateDevice((d) => ({ ...d, routines: d.routines.map((r) => r.id === id ? { ...r, enabled: !r.enabled } : r) }), "Rotina atualizada")} />}
        {view === "activity" && <ActivityView device={device} />}
        {view === "settings" && <SettingsView key={device.id} device={device} onRename={(name) => updateDevice((d) => ({ ...d, name }), "Nome atualizado")} />}
      </div>
    </main>
    <nav className="bottom-nav">{nav.slice(0, 3).map((item) => <NavItem key={item.id} active={view === item.id} icon={item.icon} label={item.label} onClick={() => setView(item.id)} />)}<NavItem active={view === "activity" || view === "settings"} icon={<MoreHorizontal size={21} />} label="Mais" onClick={() => setView("settings")} /></nav>
    {modal === "bonus" && <BonusModal onClose={() => setModal(null)} onSubmit={submitBonus} />}
    {modal === "device" && <DeviceModal onClose={() => setModal(null)} onSubmit={addComputer} />}
    {modal === "routine" && <RoutineModal onClose={() => setModal(null)} onSubmit={addRoutine} />}
    {toast && <div className="toast" role="status"><Check size={17} />{toast}</div>}
  </div>;
}

function BonusModal({ onClose, onSubmit }: { onClose: () => void; onSubmit: (value: number) => void }) {
  const [value, setValue] = useState(30);
  return <Modal title="Adicionar tempo" description="O tempo extra vale somente para hoje." onClose={onClose}><div className="preset-grid">{[15, 30, 45, 60].map((n) => <button className={value === n ? "active" : ""} key={n} onClick={() => setValue(n)}>{n}<small>min</small></button>)}</div><div className="modal-actions"><button className="button secondary" onClick={onClose}>Cancelar</button><button className="button primary" onClick={() => onSubmit(value)}><Plus size={18} />Adicionar {value}min</button></div></Modal>;
}

function DeviceModal({ onClose, onSubmit }: { onClose: () => void; onSubmit: (name: string) => void }) {
  const [name, setName] = useState("");
  return <Modal title="Adicionar computador" description="Crie o perfil antes de parear o agente." onClose={onClose}><form onSubmit={(e) => { e.preventDefault(); if (name.trim()) onSubmit(name.trim()); }}><label className="field">Nome do computador<input autoFocus placeholder="Ex.: Notebook da Ana" value={name} onChange={(e) => setName(e.target.value)} /></label><div className="modal-actions"><button className="button secondary" type="button" onClick={onClose}>Cancelar</button><button className="button primary" disabled={!name.trim()}><Plus size={18} />Adicionar</button></div></form></Modal>;
}

function RoutineModal({ onClose, onSubmit }: { onClose: () => void; onSubmit: (name: string, start: string, end: string) => void }) {
  const [name, setName] = useState("Estudos"); const [start, setStart] = useState("14:00"); const [end, setEnd] = useState("16:00");
  function submit(e: FormEvent) { e.preventDefault(); if (name.trim()) onSubmit(name.trim(), start, end); }
  return <Modal title="Nova rotina" description="Escolha um nome e o período desta rotina." onClose={onClose}><form className="routine-form" onSubmit={submit}><label className="field">Nome<input value={name} onChange={(e) => setName(e.target.value)} /></label><div className="field-row"><label className="field">Começa<input type="time" value={start} onChange={(e) => setStart(e.target.value)} /></label><label className="field">Termina<input type="time" value={end} onChange={(e) => setEnd(e.target.value)} /></label></div><div className="modal-actions"><button className="button secondary" type="button" onClick={onClose}>Cancelar</button><button className="button primary"><CalendarDays size={18} />Criar rotina</button></div></form></Modal>;
}
