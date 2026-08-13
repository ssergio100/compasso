import type { DeviceDetailResponse } from "./compassoApi";
import { APIContractError, mapDeviceDetail } from "./remoteMappers";

function detail(controlStatus: "block_requested" | "blocked", actualState: "unblocked" | "blocked"): DeviceDetailResponse {
  return {
    device: {
      id: "device-1", name: "Quarto", last_seen_at: "2026-08-10T12:00:00Z",
      policy_revision: 1, applied_policy_revision: 1, graphical_session_active: true, online: true,
    },
    policy: {
      revision: 1, monitoring_paused: false, manual_block: false, warning_minutes: 10,
      password_set: false, weekly_quota_seconds: [0, 0, 0, 0, 0, 0, 0], routines: [],
    },
    control: { revision: 2, monitoring_paused: false, manual_block: true },
    status: {
      local_date: "2026-08-10", today_quota_seconds: 0, bonus_seconds: 0, used_seconds: 0, remaining_seconds: 0,
      counting: false, online: true, graphical_session_active: true, next_block: "",
      policy_revision: 1, applied_policy_revision: 1, control_revision: 2,
      applied_control_revision: controlStatus === "blocked" ? 2 : 1,
      desired_state: "blocked", actual_state: actualState, control_status: controlStatus,
    },
    events: [],
  };
}

describe("remote control mapping", () => {
	it("reports an incompatible API contract explicitly", () => {
		const legacy = detail("blocked", "blocked") as unknown as Record<string, unknown>;
		delete legacy.control;
		expect(() => mapDeviceDetail(legacy as unknown as DeviceDetailResponse)).toThrow(APIContractError);
	});

  it("keeps requested and confirmed block states distinct", () => {
    const requested = mapDeviceDetail(detail("block_requested", "unblocked")).client;
    const confirmed = mapDeviceDetail(detail("blocked", "blocked")).client;

    expect(requested.controlStatus).toBe("block_requested");
    expect(requested.actualState).toBe("unblocked");
    expect(confirmed.controlStatus).toBe("blocked");
    expect(confirmed.actualState).toBe("blocked");
  });

  it("maps bonus granted before the first device synchronization", () => {
    const response = detail("blocked", "blocked");
    response.status.bonus_seconds = 4 * 60 * 60;
    response.status.remaining_seconds = 4 * 60 * 60;

    const mapped = mapDeviceDetail(response);

    expect(mapped.client.dailyLimitMinutes).toBe(0);
    expect(mapped.client.remainingMinutes).toBe(240);
    expect(mapped.control.bonusMinutes).toBe(240);
  });
});
