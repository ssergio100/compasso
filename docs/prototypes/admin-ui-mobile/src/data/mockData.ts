import type { AuditEvent, Client, ClientAdministrationStates, Routine, WeeklyLimits } from "../domain/models";

export const QUARTO_CLIENT_ID = "8f3d8a1c-4e2b-4c91-8d76-2f5a7b9c103e";
export const ESTUDOS_CLIENT_ID = "c2a71e64-9b30-47d5-a814-6e3f0c928b42";

export const initialClients: Client[] = [
  {
    id: QUARTO_CLIENT_ID,
    name: "Computador do quarto",
    initials: "CQ",
    agentOnline: true,
    graphicalSessionActive: true,
    monitoringActive: true,
    countingTime: true,
    policyRevision: 8,
    appliedPolicyRevision: 8,
	controlRevision: 1,
	appliedControlRevision: 1,
	actualState: "unblocked",
	controlStatus: "active",
    remainingMinutes: 102,
    usedMinutes: 68,
    dailyLimitMinutes: 170,
    lastSynchronization: "Agora mesmo",
  },
  {
    id: ESTUDOS_CLIENT_ID,
    name: "Notebook de estudos",
    initials: "NE",
    agentOnline: false,
    graphicalSessionActive: false,
    monitoringActive: false,
    countingTime: false,
    policyRevision: 5,
    appliedPolicyRevision: 5,
	controlRevision: 1,
	appliedControlRevision: 0,
	actualState: "offline",
	controlStatus: "offline",
    remainingMinutes: 120,
    usedMinutes: 60,
    dailyLimitMinutes: 180,
    lastSynchronization: "Há 18 minutos",
  },
];

export const initialLimits: WeeklyLimits = {
  [QUARTO_CLIENT_ID]: [0, 180, 180, 180, 180, 240, 240],
  [ESTUDOS_CLIENT_ID]: [120, 180, 180, 180, 180, 240, 180],
};

export const initialClientAdministration: ClientAdministrationStates = {
  [QUARTO_CLIENT_ID]: { credentialActive: true, localPasswordSet: true, warningMinutes: 10 },
  [ESTUDOS_CLIENT_ID]: { credentialActive: true, localPasswordSet: true, warningMinutes: 10 },
};

export const initialAuditEvents: AuditEvent[] = [
  {
    id: "event-quarto-sync",
    clientId: QUARTO_CLIENT_ID,
    title: "Agente sincronizado",
    detail: "Políticas e estado atualizados com sucesso.",
    createdAt: "Agora mesmo",
  },
  {
    id: "event-quarto-policy",
    clientId: QUARTO_CLIENT_ID,
    title: "Limites atualizados",
    detail: "Nova regra semanal enviada ao computador.",
    createdAt: "Hoje, 17:42",
  },
  {
    id: "event-estudos-offline",
    clientId: ESTUDOS_CLIENT_ID,
    title: "Agente desconectado",
    detail: "A última comunicação ocorreu há 18 minutos.",
    createdAt: "Hoje, 17:25",
  },
];

export const initialRoutines: Routine[] = [
  {
    id: "sleep",
    clientId: QUARTO_CLIENT_ID,
    name: "Hora de dormir",
    icon: "moon",
    start: "22:00",
    end: "07:00",
    days: [0, 1, 2, 3, 4],
    enabled: true,
  },
  {
    id: "study",
    clientId: QUARTO_CLIENT_ID,
    name: "Tempo de estudo",
    icon: "book-open",
    start: "18:30",
    end: "20:00",
    days: [1, 2, 3, 4, 5],
    enabled: true,
  },
];
