import { Check, ChevronDown, X } from "lucide-react";
import { useEffect, useRef, useState, type CSSProperties, type KeyboardEvent, type PointerEvent as ReactPointerEvent, type ReactNode } from "react";

export function Brand() {
  return <span className="brand" aria-label="Compasso"><svg aria-hidden="true" viewBox="0 0 52 32"><path d="M5 23V18M15 23V11M26 23V4M37 23V11M47 23V18" /><path className="base" d="M4 28H48" /></svg><strong>Compasso</strong></span>;
}

export function Modal({ title, description, children, onClose }: { title: string; description?: string; children: ReactNode; onClose: () => void }) {
  return <div className="modal-backdrop" onMouseDown={(event) => event.target === event.currentTarget && onClose()}><section aria-label={title} className="modal" role="dialog" aria-modal="true"><header><div><h2>{title}</h2>{description && <p>{description}</p>}</div><button aria-label="Fechar" onClick={onClose}><X size={20} /></button></header>{children}</section></div>;
}

export function Toast({ children }: { children: ReactNode }) { return <div className="toast" role="status"><Check size={17} />{children}</div>; }

function padTime(value: number) { return String(value).padStart(2, "0"); }
function normalizedTime(value: string) { const [rawHour, rawMinute] = value.split(":").map(Number); const rounded = Math.round(rawMinute / 5) * 5; return `${padTime((rawHour + Math.floor(rounded / 60)) % 24)}:${padTime(rounded % 60)}`; }
function timeMinutes(value: string) { const [hour, minute] = value.split(":").map(Number); return hour * 60 + minute; }
function minutesTime(value: number) { const minutes = (value + 1440) % 1440; return `${padTime(Math.floor(minutes / 60))}:${padTime(minutes % 60)}`; }

export function TimePicker({ label, value, onChange }: { label: string; value: string; onChange: (value: string) => void }) {
  const dialog = useRef<HTMLDialogElement>(null);
  const dragging = useRef(false);
  const [open, setOpen] = useState(false);
  const [draft, setDraft] = useState(value);
  useEffect(() => {
    if (open) {
      dialog.current?.showModal();
    } else if (dialog.current?.open) dialog.current.close();
  }, [open]);

  const updateFromPointer = (event: ReactPointerEvent<HTMLDivElement>) => {
    const bounds = event.currentTarget.getBoundingClientRect();
    const x = event.clientX - bounds.left - bounds.width / 2;
    const y = event.clientY - bounds.top - bounds.height / 2;
    if (Math.hypot(x, y) < bounds.width * .22) return;
    const angle = (Math.atan2(y, x) * 180 / Math.PI + 90 + 360) % 360;
    setDraft(minutesTime(Math.round(angle / 360 * 1440 / 5) * 5));
  };

  const minutes = timeMinutes(draft);
  const angle = minutes / 1440 * 360;
  const radians = (angle - 90) * Math.PI / 180;
  const knobStyle: CSSProperties = { left: `calc(50% + ${Math.cos(radians) * 41}%)`, top: `calc(50% + ${Math.sin(radians) * 41}%)` };
  const ringStyle: CSSProperties = { background: `conic-gradient(from 0deg, var(--ink) 0deg ${angle}deg, var(--line) ${angle}deg 360deg)` };
  const adjustWithKeyboard = (event: KeyboardEvent<HTMLDivElement>) => { if (!["ArrowLeft", "ArrowDown", "ArrowRight", "ArrowUp", "Home", "End"].includes(event.key)) return; event.preventDefault(); if (event.key === "Home") setDraft("00:00"); else if (event.key === "End") setDraft("23:55"); else setDraft(minutesTime(minutes + (["ArrowRight", "ArrowUp"].includes(event.key) ? 5 : -5))); };
  return <div className="time-field"><span>{label}</span><button aria-label={`${label}: ${value}`} onClick={() => { setDraft(normalizedTime(value)); setOpen(true); }} type="button"><strong>{value}</strong><ChevronDown size={17} /></button><dialog className="time-dialog" ref={dialog} onCancel={(event) => { event.preventDefault(); setOpen(false); }}><div className="time-sheet"><header><strong>Ajustar horário</strong><button aria-label="Fechar" onClick={() => setOpen(false)} type="button"><X size={20} /></button></header><div aria-label={`${label}: ${draft}`} aria-valuemax={1435} aria-valuemin={0} aria-valuenow={minutes} aria-valuetext={draft} className="time-ring" onKeyDown={adjustWithKeyboard} onPointerDown={(event) => { dragging.current = true; event.currentTarget.setPointerCapture(event.pointerId); updateFromPointer(event); }} onPointerMove={(event) => { if (dragging.current) updateFromPointer(event); }} onPointerUp={(event) => { dragging.current = false; event.currentTarget.releasePointerCapture(event.pointerId); }} role="slider" style={ringStyle} tabIndex={0}><div><span>{label.toUpperCase()}</span><strong>{draft}</strong></div><i aria-hidden="true" style={knobStyle} /></div><p className="ring-hint">Toque ou arraste para ajustar de 5 em 5 minutos.</p><footer><button onClick={() => setOpen(false)} type="button">Cancelar</button><button className="primary" onClick={() => { onChange(draft); setOpen(false); }} type="button"><Check size={17} /> Usar {draft}</button></footer></div></dialog></div>;
}
