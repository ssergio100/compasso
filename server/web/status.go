package web

import (
	"context"
	"time"

	"github.com/sergio/compasso/agent/policy"
	"github.com/sergio/compasso/server/storage"
)

type deviceLiveStatus struct {
	LocalDate              string `json:"local_date"`
	TodayQuotaSeconds      int64  `json:"today_quota_seconds"`
	UsedSeconds            int64  `json:"used_seconds"`
	RemainingSeconds       int64  `json:"remaining_seconds"`
	Counting               bool   `json:"counting"`
	Online                 bool   `json:"online"`
	GraphicalSessionActive bool   `json:"graphical_session_active"`
	NextBlock              string `json:"next_block"`
	PolicyRevision         int64  `json:"policy_revision"`
	AppliedPolicyRevision  int64  `json:"applied_policy_revision"`
}

func (a *App) loadDeviceLiveStatus(ctx context.Context, deviceID string) (storage.Device, storage.Policy, deviceLiveStatus, error) {
	device, storedPolicy, err := a.store.LoadDevice(ctx, deviceID)
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
		LocalDate: localDate, TodayQuotaSeconds: todayQuota, UsedSeconds: summary.UsedSeconds,
		RemainingSeconds: remaining, Online: online, NextBlock: "Aguardando sincronização",
		GraphicalSessionActive: device.GraphicalSessionActive,
		PolicyRevision:         storedPolicy.Revision, AppliedPolicyRevision: device.AppliedPolicyRevision,
	}
	decisionInput := policy.Input{
		Now: now, Quota: secondsQuota(storedPolicy.WeeklyQuota),
		ManualBlock: storedPolicy.ManualBlock,
		Consumed:    time.Duration(summary.UsedSeconds) * time.Second,
		Bonus:       time.Duration(summary.BonusSeconds) * time.Second,
	}
	if storedPolicy.MonitoringPaused {
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
