import type { Device } from "./types";

export const mockDevices: Device[] = [
  {
    id: "8f3d8a1c-4e2b-4c91-8d76-2f5a7b9c103e", name: "Computador do quarto", avatar_key: "fox", online: true, graphical_session_active: true,
    monitoring_paused: false, manual_block: false, actual_state: "unblocked", control_status: "active", counting: true, used_seconds: 68 * 60,
    remaining_seconds: 102 * 60, bonus_seconds: 0, today_quota_seconds: 170 * 60,
    warning_minutes: 10, last_seen_at: new Date().toISOString(), password_set: true,
    weekly_quota_seconds: [0, 180, 180, 180, 180, 240, 240].map((v) => v * 60),
    routines: [
      { id: "study", name: "Tempo de estudo", icon_key: "study", days: [false, true, true, true, true, true, false], start_second: 18 * 3600 + 30 * 60, end_second: 20 * 3600, enabled: true },
      { id: "sleep", name: "Hora de dormir", icon_key: "sleep", days: [true, true, true, true, true, false, false], start_second: 22 * 3600, end_second: 7 * 3600, enabled: true },
    ],
  },
  {
    id: "c2a71e64-9b30-47d5-a814-6e3f0c928b42", name: "Notebook de estudos", avatar_key: "owl", online: false, graphical_session_active: false,
    monitoring_paused: false, manual_block: false, actual_state: "offline", control_status: "offline", counting: false, used_seconds: 60 * 60,
    remaining_seconds: 120 * 60, bonus_seconds: 0, today_quota_seconds: 180 * 60,
    warning_minutes: 10, last_seen_at: new Date(Date.now() - 18 * 60_000).toISOString(), password_set: true,
    weekly_quota_seconds: [120, 180, 180, 180, 180, 240, 180].map((v) => v * 60), routines: [],
  },
];
