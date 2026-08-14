import type { AuditEvent, Client, ClientAdministrationState, ClientControlState, Routine, RoutineIconName } from "../domain/models";
import type { AuditEventResponse, DeviceDetailResponse, RoutineResponse } from "./compassoApi";

export class APIContractError extends Error {
  constructor() {
    super("A API não fornece os campos de controle exigidos por esta interface.");
    this.name = "APIContractError";
  }
}

const eventTitles: Record<string, string> = {
  device_created: "Cliente criado",
  device_renamed: "Cliente renomeado",
  quotas_updated: "Limites atualizados",
  routine_saved: "Rotina salva",
  routine_deleted: "Rotina removida",
  local_password_changed: "Senha local atualizada",
  device_token_issued: "Credencial gerada",
  device_token_revoked: "Credencial revogada",
  bonus_added: "Tempo adicional",
  pause_monitoring: "Uso pausado",
  resume_monitoring: "Uso retomado",
  block_now: "Uso bloqueado",
  clear_manual_block: "Bloqueio removido",
};

function initials(name: string) {
  const words = name.trim().split(/\s+/).filter(Boolean);
  return (words.length > 1 ? `${words[0][0]}${words.at(-1)?.[0]}` : name.slice(0, 2)).toUpperCase();
}

function routineIcon(name: string): RoutineIconName {
  const normalized = name.normalize("NFD").replace(/[\u0300-\u036f]/g, "").toLowerCase();
  if (normalized.includes("dorm") || normalized.includes("noite")) return "moon";
  if (normalized.includes("estud") || normalized.includes("leitura")) return "book-open";
  if (normalized.includes("aula")) return "graduation-cap";
  if (normalized.includes("jog")) return "gamepad";
  if (normalized.includes("refei") || normalized.includes("almoc") || normalized.includes("jantar")) return "utensils";
  return "focus";
}

function clock(seconds: number) {
  const normalized = Math.max(0, seconds);
  return `${String(Math.floor(normalized / 3600)).padStart(2, "0")}:${String(Math.floor((normalized % 3600) / 60)).padStart(2, "0")}`;
}

export function mapRoutine(deviceId: string, routine: RoutineResponse): Routine {
  return {
    id: routine.id,
    clientId: deviceId,
    name: routine.name,
    icon: routineIcon(routine.name),
    start: clock(routine.start_second),
    end: clock(routine.end_second),
    days: routine.days.flatMap((enabled, day) => enabled ? [day] : []),
    enabled: routine.enabled,
  };
}

export function mapEvent(deviceId: string, event: AuditEventResponse): AuditEvent {
  return {
    id: event.id,
    clientId: deviceId,
    title: eventTitles[event.kind] ?? event.kind,
    detail: event.details || `Origem: ${event.origin}`,
    createdAt: new Intl.DateTimeFormat("pt-BR", { dateStyle: "short", timeStyle: "short" }).format(new Date(event.created_at)),
  };
}

export function mapDeviceDetail(detail: DeviceDetailResponse): {
  administration: ClientAdministrationState;
  client: Client;
  control: ClientControlState;
  events: AuditEvent[];
  limits: number[];
  routines: Routine[];
} {
  if (!detail.control || !detail.status || typeof detail.status.control_status !== "string" ||
    typeof detail.status.actual_state !== "string" || typeof detail.status.control_revision !== "number" ||
    typeof detail.status.bonus_seconds !== "number") {
    throw new APIContractError();
  }
  const { device, policy, control, status } = detail;
  return {
    client: {
      id: device.id,
      name: device.name,
      initials: initials(device.name),
      agentOnline: status.online,
      graphicalSessionActive: status.graphical_session_active,
      monitoringActive: !control.monitoring_paused,
      countingTime: status.counting,
      policyRevision: status.policy_revision,
      appliedPolicyRevision: status.applied_policy_revision,
		controlRevision: status.control_revision,
		appliedControlRevision: status.applied_control_revision,
		actualState: status.actual_state,
		controlStatus: status.control_status,
      remainingMinutes: Math.floor(status.remaining_seconds / 60),
      usedMinutes: Math.floor(status.used_seconds / 60),
      dailyLimitMinutes: Math.floor(status.today_quota_seconds / 60),
      lastSynchronization: device.last_seen_at
        ? new Intl.DateTimeFormat("pt-BR", { dateStyle: "short", timeStyle: "short" }).format(new Date(device.last_seen_at))
        : "Ainda não sincronizado",
    },
    control: {
      paused: control.monitoring_paused,
      blocked: control.manual_block,
      bonusMinutes: Math.floor(status.bonus_seconds / 60),
    },
    administration: { credentialActive: null, localPasswordSet: policy.password_set, warningMinutes: policy.warning_minutes },
    limits: policy.weekly_quota_seconds.map((seconds) => Math.floor(seconds / 60)),
    routines: policy.routines.map((routine) => mapRoutine(device.id, routine)),
    events: detail.events.map((event) => mapEvent(device.id, event)),
  };
}
