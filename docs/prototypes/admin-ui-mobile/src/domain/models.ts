export type RoutineIconName =
  | "moon"
  | "book-open"
  | "graduation-cap"
  | "utensils"
  | "gamepad"
  | "tv"
  | "dumbbell"
  | "music"
  | "house"
  | "footprints"
  | "sparkles"
  | "focus";

export interface Client {
  id: string;
  name: string;
  initials: string;
  agentOnline: boolean;
  graphicalSessionActive: boolean;
  monitoringActive: boolean;
  countingTime: boolean;
  policyRevision: number;
  appliedPolicyRevision: number;
	controlRevision: number;
	appliedControlRevision: number;
	actualState: "offline" | "unblocked" | "blocked";
	controlStatus: "active" | "pause_requested" | "paused" | "block_requested" | "blocked" | "offline";
  remainingMinutes: number;
  usedMinutes: number;
  dailyLimitMinutes: number;
  lastSynchronization: string;
}

export interface ClientDraft {
  name: string;
}

export interface ClientControlState {
  paused: boolean;
  blocked: boolean;
  bonusMinutes: number;
}

export type ClientControlStates = Record<string, ClientControlState>;

export interface ClientAdministrationState {
  credentialActive: boolean | null;
  localPasswordSet: boolean;
  warningMinutes: number;
}

export type ClientAdministrationStates = Record<string, ClientAdministrationState>;

export type OperationStatus = "pending" | "applied" | "error";

export interface ClientOperation {
  id: string;
  clientId: string;
  label: string;
  status: OperationStatus;
  detail: string;
	confirmation: "policy" | "bonus" | "pause" | "resume" | "block" | "unblock";
}

export type ClientOperations = Record<string, ClientOperation>;

export interface AuditEvent {
  id: string;
  clientId: string;
  title: string;
  detail: string;
  createdAt: string;
}

export interface Routine {
  id: string;
  clientId: string;
  name: string;
  icon: RoutineIconName;
  start: string;
  end: string;
  days: number[];
  enabled: boolean;
}

export interface RoutineDraft {
  name: string;
  icon: RoutineIconName;
  start: string;
  end: string;
  days: number[];
}

export type WeeklyLimits = Record<string, number[]>;

export interface DataLoadError {
  title: string;
  description: string;
}
