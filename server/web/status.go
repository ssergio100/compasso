package web

import (
	"context"
	"time"

	"github.com/ssergio100/compasso/agent/policy"
	"github.com/ssergio100/compasso/server/storage"
)

type deviceLiveStatus struct {
	LocalDate              string `json:"local_date"`
	TodayQuotaSeconds      int64  `json:"today_quota_seconds"`
	BonusSeconds           int64  `json:"bonus_seconds"`
	UsedSeconds            int64  `json:"used_seconds"`
	RemainingSeconds       int64  `json:"remaining_seconds"`
	Counting               bool   `json:"counting"`
	Online                 bool   `json:"online"`
	GraphicalSessionActive bool   `json:"graphical_session_active"`
	NextBlock              string `json:"next_block"`
	PolicyRevision         int64  `json:"policy_revision"`
	AppliedPolicyRevision  int64  `json:"applied_policy_revision"`
	ControlRevision        int64  `json:"control_revision"`
	AppliedControlRevision int64  `json:"applied_control_revision"`
	DesiredState           string `json:"desired_state"`
	ActualState            string `json:"actual_state"`
	ControlStatus          string `json:"control_status"`
}

func (a *App) loadDeviceLiveStatus(ctx context.Context, deviceID string) (storage.Device, storage.Policy, deviceLiveStatus, error) {
	device, storedPolicy, err := a.store.LoadDevice(ctx, deviceID)
	if err != nil {
		return storage.Device{}, storage.Policy{}, deviceLiveStatus{}, err
	}
	control, err := a.store.LoadControl(ctx, deviceID)
	if err != nil {
		return storage.Device{}, storage.Policy{}, deviceLiveStatus{}, err
	}
	pendingControlKind, err := a.store.PendingControlKind(ctx, deviceID)
	if err != nil {
		return storage.Device{}, storage.Policy{}, deviceLiveStatus{}, err
	}
	now := a.now()
	localDate := now.Format("2006-01-02")
	online := isOnline(device.LastSeenAt, now, a.onlineTimeout)
	if online {
		latestHeartbeatLocalDate, dateErr := a.store.LatestHeartbeatLocalDate(ctx, deviceID)
		if dateErr != nil {
			return storage.Device{}, storage.Policy{}, deviceLiveStatus{}, dateErr
		}
		if latestHeartbeatLocalDate != "" {
			localDate = latestHeartbeatLocalDate
		}
	}
	summary, err := a.store.LoadDailySummary(ctx, deviceID, localDate)
	if err != nil {
		return storage.Device{}, storage.Policy{}, deviceLiveStatus{}, err
	}
	unacknowledgedBonus, err := a.store.UnacknowledgedRemoteBonusSeconds(ctx, deviceID, localDate)
	if err != nil {
		return storage.Device{}, storage.Policy{}, deviceLiveStatus{}, err
	}
	summary.BonusSeconds -= unacknowledgedBonus
	if summary.BonusSeconds < 0 {
		summary.BonusSeconds = 0
	}
	localDay, err := time.Parse("2006-01-02", localDate)
	if err != nil {
		return storage.Device{}, storage.Policy{}, deviceLiveStatus{}, err
	}
	todayQuota := storedPolicy.WeeklyQuota[localDay.Weekday()]
	remaining := todayQuota + summary.BonusSeconds - summary.UsedSeconds
	if remaining < 0 {
		remaining = 0
	}
	liveStatus := deviceLiveStatus{
		LocalDate: localDate, TodayQuotaSeconds: todayQuota, BonusSeconds: summary.BonusSeconds, UsedSeconds: summary.UsedSeconds,
		RemainingSeconds: remaining, Online: online, NextBlock: "Aguardando sincronização",
		GraphicalSessionActive: device.GraphicalSessionActive,
		PolicyRevision:         storedPolicy.Revision, AppliedPolicyRevision: device.AppliedPolicyRevision,
		ControlRevision: control.Revision, AppliedControlRevision: device.AppliedControlRevision,
	}
	switch {
	case !online:
		liveStatus.ActualState = "offline"
	case device.GraphicalSessionLocked:
		liveStatus.ActualState = "blocked"
	default:
		liveStatus.ActualState = "unblocked"
	}
	switch {
	case control.MonitoringPaused:
		liveStatus.DesiredState = "paused"
	case control.ManualBlock:
		liveStatus.DesiredState = "blocked"
	default:
		liveStatus.DesiredState = "policy"
	}
	switch {
	case !online:
		liveStatus.ControlStatus = "offline"
	case pendingControlKind == "pause_monitoring":
		liveStatus.ControlStatus = "pause_requested"
	case pendingControlKind == "resume_monitoring":
		liveStatus.ControlStatus = "resume_requested"
	case pendingControlKind == "block_now":
		liveStatus.ControlStatus = "block_requested"
	case pendingControlKind == "clear_manual_block":
		liveStatus.ControlStatus = "unblock_requested"
	case control.MonitoringPaused:
		liveStatus.ControlStatus = "paused"
	case control.ManualBlock && (!device.GraphicalSessionLocked || device.AppliedControlRevision < control.Revision):
		liveStatus.ControlStatus = "block_requested"
	case control.ManualBlock:
		liveStatus.ControlStatus = "blocked"
	case device.GraphicalSessionLocked:
		liveStatus.ControlStatus = "blocked"
	default:
		liveStatus.ControlStatus = "active"
	}
	decisionInput := policy.Input{
		Now: now, Quota: secondsQuota(storedPolicy.WeeklyQuota),
		ManualBlock: control.ManualBlock,
		Consumed:    time.Duration(summary.UsedSeconds) * time.Second,
		Bonus:       time.Duration(summary.BonusSeconds) * time.Second,
	}
	if control.MonitoringPaused {
		decisionInput.Monitoring = policy.MonitoringPaused
	}
	for _, routine := range storedPolicy.Routines {
		if !routine.Enabled {
			continue
		}
		decisionInput.Routines = append(decisionInput.Routines, policy.Routine{
			Name: routine.Name, Days: routine.Days,
			Start: time.Duration(routine.Start) * time.Second, End: time.Duration(routine.End) * time.Second,
		})
	}
	if decision, evaluationErr := policy.Evaluate(decisionInput); evaluationErr == nil {
		liveStatus.Counting = decision.ShouldCount && online && device.GraphicalSessionActive
		if !decision.NextBlockAt.IsZero() {
			liveStatus.NextBlock = decision.NextBlockAt.Format("02/01 15:04")
		}
	}
	return device, storedPolicy, liveStatus, nil
}

func isOnline(lastSeen *time.Time, now time.Time, timeout time.Duration) bool {
	return lastSeen != nil && !lastSeen.Before(now.Add(-timeout))
}

func secondsQuota(stored [7]int64) (converted policy.WeeklyQuota) {
	for day, seconds := range stored {
		converted[day] = time.Duration(seconds) * time.Second
	}
	return converted
}
