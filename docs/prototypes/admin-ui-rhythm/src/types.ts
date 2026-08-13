export type View = "now" | "limits" | "routines" | "administration";

export interface Routine {
  id: string;
  name: string;
  days: [boolean, boolean, boolean, boolean, boolean, boolean, boolean];
  start_second: number;
  end_second: number;
  enabled: boolean;
}

export interface Device {
  id: string;
  name: string;
  online: boolean;
  graphical_session_active: boolean;
  monitoring_paused: boolean;
  manual_block: boolean;
  counting: boolean;
  used_seconds: number;
  remaining_seconds: number;
  bonus_seconds: number;
  today_quota_seconds: number;
  warning_minutes: number;
  last_seen_at: string | null;
  weekly_quota_seconds: number[];
  routines: Routine[];
  password_set: boolean;
}

export interface Session { authenticated: boolean; login?: string; csrf_token: string; setup_required: boolean }

export interface DeviceResponse {
  id: string; name: string; last_seen_at: string | null; policy_revision: number; applied_policy_revision: number;
  graphical_session_active: boolean; online: boolean;
}

export interface DeviceDetailResponse {
  device: DeviceResponse;
  policy: { revision: number; monitoring_paused: boolean; manual_block: boolean; warning_minutes: number; password_set: boolean; weekly_quota_seconds: number[]; routines: Routine[] };
  control: { revision: number; monitoring_paused: boolean; manual_block: boolean };
  status: { today_quota_seconds: number; bonus_seconds: number; used_seconds: number; remaining_seconds: number; counting: boolean; online: boolean; graphical_session_active: boolean };
}
