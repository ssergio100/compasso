export type View = "home" | "limits" | "routines" | "activity" | "settings";

export interface Device {
  id: string;
  name: string;
  online: boolean;
  inUse: boolean;
  paused: boolean;
  blocked: boolean;
  usedMinutes: number;
  limitMinutes: number;
  bonusMinutes: number;
  lastSeen: string;
  limits: number[];
  routines: Routine[];
  events: EventItem[];
  warningMinutes: number;
}

export interface Routine {
  id: string;
  name: string;
  start: string;
  end: string;
  days: number[];
  enabled: boolean;
  kind: "study" | "free" | "sleep";
}

export interface EventItem {
  id: string;
  title: string;
  detail: string;
  time: string;
  tone: "green" | "blue" | "orange";
}
