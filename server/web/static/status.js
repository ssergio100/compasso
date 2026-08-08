(() => {
  const successNotice = document.querySelector(".notice.success");
  if (successNotice) {
    const currentURL = new URL(window.location.href);
    currentURL.searchParams.delete("success");
    window.history.replaceState({}, "", currentURL);
    window.setTimeout(() => successNotice.remove(), 5000);
  }

  const remainingTime = document.getElementById("remaining-time");
  if (!remainingTime) return;

  const usedTime = document.getElementById("used-time");

  let remainingSeconds = Number.parseInt(remainingTime.dataset.remainingSeconds, 10);
  if (!Number.isFinite(remainingSeconds) || remainingSeconds < 0) return;

  let usedSeconds = usedTime
    ? Number.parseInt(usedTime.dataset.usedSeconds, 10)
    : 0;
  if (!Number.isFinite(usedSeconds) || usedSeconds < 0) usedSeconds = 0;

  const formatDuration = (durationSeconds) => {
    const hours = Math.floor(durationSeconds / 3600);
    const minutes = Math.floor((durationSeconds % 3600) / 60);
    const seconds = durationSeconds % 60;
    return [hours, minutes, seconds]
      .map((value) => String(value).padStart(2, "0"))
      .join(":");
  };

  const renderCounters = () => {
    remainingTime.textContent = formatDuration(remainingSeconds);
    if (usedTime) usedTime.textContent = formatDuration(usedSeconds);
  };

  let isCounting = remainingTime.dataset.counting === "true";
  renderCounters();

  window.setInterval(() => {
    if (isCounting && remainingSeconds > 0) {
      remainingSeconds -= 1;
      usedSeconds += 1;
    }
    renderCounters();
  }, 1000);

  const synchronizeCounters = async () => {
    try {
      const response = await window.fetch(remainingTime.dataset.statusUrl, {
        cache: "no-store",
        headers: { Accept: "application/json" },
      });
      if (!response.ok) return;

      const currentStatus = await response.json();
      const refreshedRemainingSeconds = Number.parseInt(currentStatus.remaining_seconds, 10);
      const refreshedUsedSeconds = Number.parseInt(currentStatus.used_seconds, 10);
      if (!Number.isFinite(refreshedRemainingSeconds) || !Number.isFinite(refreshedUsedSeconds)) return;

      remainingSeconds = Math.max(0, refreshedRemainingSeconds);
      usedSeconds = Math.max(0, refreshedUsedSeconds);
      isCounting = currentStatus.counting === true;
      renderCounters();
    } catch (_error) {
      // Keep the local counters running during a transient network failure.
    }
  };

  window.setInterval(synchronizeCounters, 2000);
})();
