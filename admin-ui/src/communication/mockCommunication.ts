import type { CommunicationLog } from "../types";

const now = Date.now();

export const mockCommunication: CommunicationLog[] = [
  [9, "agent", "api", "heartbeat", "success", 200, 84, "Heartbeat processado e resposta enviada ao agente."],
  [8, "interface", "api", "GET device", "success", 200, 112, "Estado do computador atualizado na interface."],
  [7, "api", "agent", "policy_response", "success", 200, 96, "Política enviada ao agente na revisão 124."],
  [6, "interface", "api", "POST bonus", "success", 202, 78, "Bônus registrado e aguardando confirmação do agente."],
  [5, "agent", "api", "bonus_acknowledged", "success", 200, 91, "Bônus reconhecido pelo agente após persistência."],
  [4, "agent", "api", "heartbeat", "error", 503, 1452, "API temporariamente indisponível; nova tentativa programada."],
  [3, "agent", "api", "heartbeat", "success", 200, 148, "Sincronização concluída após nova tentativa."],
  [2, "interface", "api", "PUT policy", "success", 200, 126, "Limites e antecedência do aviso atualizados."],
  [1, "agent", "api", "session_state", "warning", 409, 63, "Âncora local estava desatualizada e foi solicitada novamente."],
].map(([id, source, target, operation, result, status, duration, summary], index) => ({
  id: id as number,
  device_id: "device-bedroom",
  source: source as CommunicationLog["source"],
  target: target as CommunicationLog["target"],
  operation: operation as string,
  result: result as CommunicationLog["result"],
  http_status: status as number,
  duration_ms: duration as number,
  summary: summary as string,
  details: {
    correlation_id: `demo-${String(id).padStart(4, "0")}`,
    protocol_version: "2",
    local_date: "2026-08-14",
    policy_revision: "124",
    route: operation === "heartbeat" ? "/api/v1/device/heartbeat" : "/api/v1/admin/devices/{device_id}",
  },
  created_at: new Date(now - index * 2700).toISOString(),
}));
