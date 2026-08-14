import type { Device } from "./types";

export const initialDevices: Device[] = [
  {
    id: "clara",
    name: "Notebook da Clara",
    online: true,
    inUse: true,
    paused: false,
    blocked: false,
    usedMinutes: 58,
    limitMinutes: 160,
    bonusMinutes: 0,
    lastSeen: "agora",
    limits: [160, 160, 160, 160, 180, 180, 120],
    warningMinutes: 10,
    routines: [
      { id: "r1", name: "Estudos", start: "14:00", end: "16:00", days: [1, 2, 3, 4, 5], enabled: true, kind: "study" },
      { id: "r2", name: "Tempo livre", start: "18:30", end: "20:00", days: [1, 2, 3, 4, 5], enabled: true, kind: "free" },
      { id: "r3", name: "Hora de dormir", start: "22:00", end: "07:00", days: [0, 1, 2, 3, 4, 5, 6], enabled: true, kind: "sleep" },
    ],
    events: [
      { id: "e1", title: "Rotina iniciada: Estudos", detail: "Iniciada por Sérgio", time: "14:00", tone: "green" },
      { id: "e2", title: "Tempo adicionado", detail: "+30min por Sérgio", time: "13:10", tone: "blue" },
      { id: "e3", title: "Contagem retomada", detail: "Retomada por Sérgio", time: "12:42", tone: "orange" },
    ],
  },
  {
    id: "miguel", name: "PC do Miguel", online: true, inUse: false, paused: false, blocked: false,
    usedMinutes: 24, limitMinutes: 120, bonusMinutes: 0, lastSeen: "há 1 min", limits: [120, 120, 120, 120, 120, 180, 180], warningMinutes: 15, routines: [], events: [],
  },
  {
    id: "sala", name: "Desktop da Sala", online: false, inUse: false, paused: false, blocked: false,
    usedMinutes: 0, limitMinutes: 180, bonusMinutes: 0, lastSeen: "ontem, 22:18", limits: [180, 180, 180, 180, 180, 240, 240], warningMinutes: 10, routines: [], events: [],
  },
];
