(() => {
  "use strict";

  const application = document.getElementById("application");
  const topbar = document.getElementById("topbar");
  const adminLogin = document.getElementById("admin-login");
  const logoutButton = document.getElementById("logout-button");
  const weekdayNames = ["Domingo", "Segunda", "Terça", "Quarta", "Quinta", "Sexta", "Sábado"];
  const shortWeekdayNames = ["Dom", "Seg", "Ter", "Qua", "Qui", "Sex", "Sáb"];
  const eventNames = {
    device_created: "Dispositivo criado", device_renamed: "Dispositivo renomeado",
    quotas_updated: "Cotas atualizadas", routine_saved: "Rotina salva",
    routine_deleted: "Rotina excluída", local_password_changed: "Senha local alterada",
    device_token_issued: "Credencial do agente gerada", device_token_revoked: "Credencial revogada",
    bonus_added: "Tempo extra adicionado", pause_monitoring: "Vigilância pausada",
    resume_monitoring: "Vigilância retomada", block_now: "Bloqueio imediato",
    clear_manual_block: "Bloqueio removido",
  };

  const configuredAPIBaseURL = window.COMPASSO_CONFIG && window.COMPASSO_CONFIG.apiBaseURL;
  const api = new window.CompassoAPI.CompassoAPIClient(configuredAPIBaseURL);
  let currentSession = null;
  let currentDevice = null;
  let issuedCredential = null;
  let counterTimer = null;
  let synchronizationTimer = null;

  const escapeHTML = (value) => String(value ?? "").replace(/[&<>'"]/g, (character) => ({
    "&": "&amp;", "<": "&lt;", ">": "&gt;", "'": "&#39;", '"': "&quot;",
  })[character]);

  const formatDuration = (seconds, includeSeconds = false) => {
    const safeSeconds = Math.max(0, Number(seconds) || 0);
    const parts = [Math.floor(safeSeconds / 3600), Math.floor((safeSeconds % 3600) / 60)];
    if (includeSeconds) parts.push(Math.floor(safeSeconds % 60));
    return parts.map((part) => String(part).padStart(2, "0")).join(":");
  };

  const formatClock = (seconds) => formatDuration(seconds, false);
  const parseDuration = (value) => {
    const match = /^(\d{1,2}):([0-5]\d)$/.exec(value);
    if (!match) throw new Error("Use o formato HH:MM.");
    const hours = Number(match[1]);
    const minutes = Number(match[2]);
    if (hours > 24 || (hours === 24 && minutes !== 0)) throw new Error("A cota deve ficar entre 00:00 e 24:00.");
    return hours * 3600 + minutes * 60;
  };

  const formatLastSeen = (value) => value
    ? new Intl.DateTimeFormat("pt-BR", { dateStyle: "short", timeStyle: "medium" }).format(new Date(value))
    : "Nunca conectado";

  const connectionLabel = (status) => {
    if (!status.online) return "OFFLINE";
    return status.graphical_session_active
      ? "AGENTE ONLINE · SESSÃO ATIVA"
      : "AGENTE ONLINE · SEM SESSÃO";
  };

  const stopLiveUpdates = () => {
    window.clearInterval(counterTimer);
    window.clearInterval(synchronizationTimer);
    counterTimer = null;
    synchronizationTimer = null;
  };

  const showTopbar = (visible) => {
    topbar.hidden = !visible;
    adminLogin.textContent = visible && currentSession ? currentSession.login : "";
  };

  const noticeMarkup = (message, kind = "success") => message
    ? `<div class="notice ${kind}" role="${kind === "error" ? "alert" : "status"}">${escapeHTML(message)}</div>`
    : "";

  const scheduleSuccessNoticeDismissal = () => {
    const successNotice = application.querySelector(".notice.success");
    if (!successNotice) return;
    window.setTimeout(() => successNotice.remove(), 4000);
  };

  const bindNavigation = () => {
    document.querySelectorAll("[data-route]").forEach((link) => {
      link.addEventListener("click", (event) => {
        event.preventDefault();
        navigate(link.getAttribute("href"));
      });
    });
  };

  const navigate = (path) => {
    window.history.pushState({}, "", path);
    route();
  };

  const handleError = (error) => {
    if (error && error.status === 401) {
      currentSession = null;
      renderLogin("Sua sessão expirou. Entre novamente.");
      return;
    }
    const message = error instanceof Error ? error.message : "Não foi possível concluir a operação.";
    const existingNotice = application.querySelector(".notice");
    if (existingNotice) existingNotice.remove();
    application.insertAdjacentHTML("afterbegin", noticeMarkup(message, "error"));
  };

  const renderLogin = (errorMessage = "") => {
    stopLiveUpdates();
    showTopbar(false);
    application.innerHTML = `${noticeMarkup(errorMessage, "error")}
      <section class="login-card card">
        <p class="eyebrow">CONTROLE DE TEMPO</p>
        <h1>Entrar no Compasso</h1>
        <p class="muted">Acesse as configurações do computador controlado.</p>
        <form id="login-form" class="stack">
          <label>Usuário<input name="login" autocomplete="username" required autofocus></label>
          <label>Senha<input type="password" name="password" autocomplete="current-password" required></label>
          <button class="primary" type="submit">Entrar</button>
        </form>
      </section>`;
    document.getElementById("login-form").addEventListener("submit", async (event) => {
      event.preventDefault();
      const form = new FormData(event.currentTarget);
      try {
        currentSession = await api.login(form.get("login"), form.get("password"));
        navigate("/");
      } catch (error) {
        renderLogin(error.status === 401 ? "Usuário ou senha inválidos." : error.message);
      }
    });
  };

  const renderInitialSetup = (errorMessage = "") => {
    stopLiveUpdates();
    showTopbar(false);
    application.innerHTML = `${noticeMarkup(errorMessage, "error")}
      <section class="login-card card">
        <p class="eyebrow">PRIMEIRO ACESSO</p>
        <h1>Configurar o Compasso</h1>
        <p class="muted">Crie o acesso administrativo. Esta etapa aparece somente uma vez.</p>
        <form id="initial-setup-form" class="stack">
          <label>Usuário<input name="login" maxlength="80" autocomplete="username" required autofocus></label>
          <label>Senha<input type="password" name="password" autocomplete="new-password" required></label>
          <label>Confirmar senha<input type="password" name="password_confirmation" autocomplete="new-password" required></label>
          <button class="primary" type="submit">Concluir configuração</button>
        </form>
      </section>`;
    document.getElementById("initial-setup-form").addEventListener("submit", async (event) => {
      event.preventDefault();
      const form = new FormData(event.currentTarget);
      const password = form.get("password");
      const passwordConfirmation = form.get("password_confirmation");
      if (password !== passwordConfirmation) {
        renderInitialSetup("As senhas não coincidem.");
        return;
      }
      try {
        currentSession = await api.completeInitialSetup(form.get("login"), password, passwordConfirmation);
        window.history.replaceState({}, "", "/");
        route();
      } catch (error) {
        if (error.status === 409) {
          currentSession = await api.loadSession();
          renderLogin("A configuração inicial já foi concluída. Entre com o acesso criado.");
          return;
        }
        renderInitialSetup(error.message);
      }
    });
  };

  const loadDevices = async (message = "") => {
    stopLiveUpdates();
    currentDevice = null;
    issuedCredential = null;
    showTopbar(true);
    try {
      const response = await api.listDevices();
      const devicesMarkup = response.devices.length
        ? response.devices.map((device) => `<a class="card device-card" href="/devices/${escapeHTML(device.id)}" data-route>
            <span class="status ${device.online ? "online" : "offline"}">${connectionLabel(device)}</span>
            <h2>${escapeHTML(device.name)}</h2>
            <p>Política revisão ${device.policy_revision}</p>
            <p class="muted">Aplicada pelo cliente: revisão ${device.applied_policy_revision}</p>
          </a>`).join("")
        : `<div class="card empty-state"><h2>Nenhum computador</h2><p>Cadastre o primeiro computador para definir suas regras.</p></div>`;
      application.innerHTML = `${noticeMarkup(message)}
        <div class="page-heading"><div><p class="eyebrow">ADMINISTRAÇÃO</p><h1>Dispositivos</h1>
          <p class="muted">Computadores administrados pelo Compasso.</p></div></div>
        <div class="device-grid">${devicesMarkup}</div>
        <section class="card compact-card"><h2>Adicionar dispositivo</h2>
          <form id="create-device-form" class="inline-form">
            <label>Nome<input name="name" maxlength="80" placeholder="Ex.: PC do quarto" required></label>
            <button class="primary" type="submit">Adicionar</button>
          </form>
        </section>`;
      scheduleSuccessNoticeDismissal();
      bindNavigation();
      document.getElementById("create-device-form").addEventListener("submit", async (event) => {
        event.preventDefault();
        const name = new FormData(event.currentTarget).get("name");
        try {
          const device = await api.createDevice(name);
          navigate(`/devices/${device.id}`);
        } catch (error) { handleError(error); }
      });
    } catch (error) { handleError(error); }
  };

  const quotaInputsMarkup = (weeklyQuota) => weekdayNames.map((name, day) => `
    <label>${name}<input name="quota_${day}" value="${formatDuration(weeklyQuota[day])}" pattern="[0-9]{1,2}:[0-5][0-9]" required></label>
  `).join("");

  const routinesMarkup = (routines) => routines.length ? routines.map((routine) => {
    const selectedDays = routine.days.map((enabled, day) => enabled ? shortWeekdayNames[day] : "").filter(Boolean).join(", ");
    return `<article class="routine-row">
      <div><strong>${escapeHTML(routine.name)}</strong><span>${selectedDays} · ${formatClock(routine.start_second)}–${formatClock(routine.end_second)}</span></div>
      <div class="row-actions">
        <button class="secondary" type="button" data-edit-routine="${escapeHTML(routine.id)}">Editar</button>
        <button class="danger-text" type="button" data-delete-routine="${escapeHTML(routine.id)}">Excluir</button>
      </div>
    </article>`;
  }).join("") : `<p class="muted">Nenhuma rotina cadastrada.</p>`;

  const eventsMarkup = (events) => events.length ? events.map((event) => `<article>
    <time>${formatLastSeen(event.created_at)}</time><div><strong>${escapeHTML(eventNames[event.kind] || event.kind)}</strong>
    <span>${escapeHTML(event.origin)} · ${escapeHTML(event.details)}</span></div></article>`).join("")
    : `<p class="muted">Nenhum evento registrado.</p>`;

  const credentialMarkup = () => issuedCredential ? `<div class="notice success">Copie agora. Este token não será mostrado novamente.</div>
    <dl class="credential-grid"><div><dt>device_id</dt><dd><code>${escapeHTML(issuedCredential.device_id)}</code></dd></div>
    <div><dt>device_token</dt><dd><code>${escapeHTML(issuedCredential.device_token)}</code></dd></div></dl>` : "";

  const renderDevice = (detail, message = "") => {
    currentDevice = detail;
    const { device, policy, status, events } = detail;
    const monitoringCommand = policy.monitoring_paused ? "resume_monitoring" : "pause_monitoring";
    const monitoringLabel = policy.monitoring_paused ? "Retomar vigilância" : "Pausar vigilância";
    const blockCommand = policy.manual_block ? "clear_manual_block" : "block_now";
    const blockLabel = policy.manual_block ? "Remover bloqueio" : "Bloquear agora";
    const blockClass = policy.manual_block ? "secondary" : "danger";
    const dayCheckboxes = weekdayNames.map((name, day) => `<label><input type="checkbox" name="day_${day}"> ${name}</label>`).join("");

    application.innerHTML = `${noticeMarkup(message)}
      <nav class="breadcrumbs"><a href="/" data-route>Dispositivos</a> / ${escapeHTML(device.name)}</nav>
      <div class="page-heading"><div><p class="eyebrow">AGORA</p><h1>${escapeHTML(device.name)}</h1></div>
        <div class="connection-state"><span id="connection-status" class="status ${status.online ? "online" : "offline"}">${connectionLabel(status)}</span>
        <small>${formatLastSeen(device.last_seen_at)}</small></div></div>
      <section class="card action-panel"><div><p class="eyebrow">AÇÕES IMEDIATAS</p>
        <p class="muted">Política aplicada: revisão ${device.applied_policy_revision} de ${policy.revision}.</p></div>
        <div class="action-grid">
          <form id="bonus-form" class="inline-form"><label>Tempo extra (min)<input type="number" name="minutes" min="1" max="720" value="30" required></label>
            <button class="primary" type="submit">Adicionar</button></form>
          <button class="secondary" type="button" data-command="${monitoringCommand}">${monitoringLabel}</button>
          <button class="${blockClass}" type="button" data-command="${blockCommand}">${blockLabel}</button>
        </div></section>
      <section class="summary-grid">
        <div class="card metric featured"><span>Tempo restante</span><strong id="remaining-time">${formatDuration(status.remaining_seconds, true)}</strong></div>
        <div class="card metric"><span>Cota de hoje</span><strong id="today-quota">${formatDuration(status.today_quota_seconds)}</strong></div>
        <div class="card metric"><span>Usado hoje</span><strong id="used-time">${formatDuration(status.used_seconds, true)}</strong></div>
        <div class="card metric"><span>Próximo bloqueio</span><strong id="next-block" class="small-metric">${escapeHTML(policy.monitoring_paused ? "Vigilância pausada" : policy.manual_block ? "Bloqueio manual ativo" : status.next_block)}</strong></div>
      </section>
      <section class="card"><div class="section-heading"><div><p class="eyebrow">CONFIGURAÇÃO SEMANAL</p><h2>Cotas</h2></div>
        <span class="revision">Revisão ${policy.revision}</span></div>
        <form id="policy-form" class="stack"><div class="quota-grid">${quotaInputsMarkup(policy.weekly_quota_seconds)}</div>
          <label class="short-field">Aviso principal (minutos)<input type="number" name="warning_minutes" min="0" max="120" value="${policy.warning_minutes}" required></label>
          <button class="primary" type="submit">Salvar cotas</button></form></section>
      <section class="card"><p class="eyebrow">PAREAMENTO</p><h2>Credencial do agente</h2>${credentialMarkup()}
        <p class="muted">Gerar uma credencial invalida imediatamente qualquer token anterior.</p>
        <div class="row-actions"><button id="issue-token" class="secondary" type="button">Gerar novo token</button>
          <button id="revoke-token" class="danger-text" type="button">Revogar token atual</button></div></section>
      <section class="card"><p class="eyebrow">BLOQUEIOS RECORRENTES</p><h2>Rotinas</h2>
        <div class="routine-list">${routinesMarkup(policy.routines)}</div>
        <form id="routine-form" class="stack divided"><input type="hidden" name="routine_id"><h3 id="routine-form-title">Nova rotina</h3>
          <label>Nome<input name="name" maxlength="80" placeholder="Ex.: Hora de dormir" required></label>
          <fieldset><legend>Dias</legend><div class="day-picker">${dayCheckboxes}</div></fieldset>
          <div class="two-columns"><label>Início<input type="time" name="start" value="22:00" required></label>
            <label>Fim<input type="time" name="end" value="08:00" required></label></div>
          <div class="row-actions"><button class="primary" type="submit">Salvar rotina</button>
            <button id="cancel-routine-edit" class="secondary" type="button" hidden>Cancelar edição</button></div></form></section>
      <section class="card"><p class="eyebrow">SEGURANÇA LOCAL</p><h2>Senha do responsável</h2>
        <p class="muted">A senha atual nunca é exibida. Status: ${policy.password_set ? "configurada" : "não configurada"}.</p>
        <form id="password-form" class="stack"><div class="two-columns"><label>Nova senha<input type="password" name="password" autocomplete="new-password" required></label>
          <label>Confirmar senha<input type="password" name="password_confirmation" autocomplete="new-password" required></label></div>
          <button class="primary" type="submit">Alterar senha local</button></form></section>
      <section class="card"><p class="eyebrow">AUDITORIA</p><h2>Histórico</h2><div class="history">${eventsMarkup(events)}</div></section>
      <section class="card danger-zone"><h2>Dispositivo</h2><form id="rename-form" class="inline-form">
        <label>Nome<input name="name" maxlength="80" value="${escapeHTML(device.name)}" required></label>
        <button class="secondary" type="submit">Renomear</button></form>
        <button id="delete-device" class="danger" type="button">Excluir dispositivo</button></section>`;
    scheduleSuccessNoticeDismissal();
    bindNavigation();
    bindDeviceActions();
    startLiveUpdates();
  };

  const refreshDevice = async (message = "") => {
    const detail = await api.loadDevice(currentDevice.device.id);
    renderDevice(detail, message);
  };

  const bindDeviceActions = () => {
    const deviceID = currentDevice.device.id;
    const runAndRefresh = async (operation, message) => {
      try { await operation(); await refreshDevice(message); } catch (error) { handleError(error); }
    };

    document.getElementById("bonus-form").addEventListener("submit", (event) => {
      event.preventDefault();
      const minutes = Number(new FormData(event.currentTarget).get("minutes"));
      runAndRefresh(() => api.addBonus(deviceID, minutes), "Tempo extra enviado; aguardando sincronização.");
    });
    document.querySelectorAll("[data-command]").forEach((button) => button.addEventListener("click", () => {
      runAndRefresh(() => api.queueCommand(deviceID, button.dataset.command), "Comando enviado; será aplicado no próximo heartbeat.");
    }));
    document.getElementById("policy-form").addEventListener("submit", (event) => {
      event.preventDefault();
      try {
        const form = new FormData(event.currentTarget);
        const weeklyQuota = weekdayNames.map((_name, day) => parseDuration(form.get(`quota_${day}`)));
        runAndRefresh(() => api.updatePolicy(deviceID, weeklyQuota, Number(form.get("warning_minutes"))), "Cotas atualizadas; aguardando sincronização.");
      } catch (error) { handleError(error); }
    });
    document.getElementById("issue-token").addEventListener("click", async () => {
      if (!window.confirm("Gerar uma nova credencial e invalidar a anterior?")) return;
      try { issuedCredential = await api.issueToken(deviceID); await refreshDevice("Credencial gerada."); } catch (error) { handleError(error); }
    });
    document.getElementById("revoke-token").addEventListener("click", () => {
      if (!window.confirm("Revogar a credencial atual do agente?")) return;
      issuedCredential = null;
      runAndRefresh(() => api.revokeToken(deviceID), "Credencial revogada.");
    });

    const resetRoutineForm = () => {
      const form = document.getElementById("routine-form");
      form.reset();
      form.elements.routine_id.value = "";
      form.elements.start.value = "22:00";
      form.elements.end.value = "08:00";
      document.getElementById("routine-form-title").textContent = "Nova rotina";
      document.getElementById("cancel-routine-edit").hidden = true;
    };
    document.querySelectorAll("[data-edit-routine]").forEach((button) => button.addEventListener("click", () => {
      const routine = currentDevice.policy.routines.find((item) => item.id === button.dataset.editRoutine);
      if (!routine) return;
      const form = document.getElementById("routine-form");
      form.elements.routine_id.value = routine.id;
      form.elements.name.value = routine.name;
      form.elements.start.value = formatClock(routine.start_second);
      form.elements.end.value = formatClock(routine.end_second);
      routine.days.forEach((enabled, day) => { form.elements[`day_${day}`].checked = enabled; });
      document.getElementById("routine-form-title").textContent = "Editar rotina";
      document.getElementById("cancel-routine-edit").hidden = false;
      form.scrollIntoView({ behavior: "smooth" });
    }));
    document.getElementById("cancel-routine-edit").addEventListener("click", resetRoutineForm);
    document.getElementById("routine-form").addEventListener("submit", (event) => {
      event.preventDefault();
      const form = new FormData(event.currentTarget);
      const routine = {
        name: form.get("name"), days: weekdayNames.map((_name, day) => form.get(`day_${day}`) === "on"),
        start_second: parseDuration(form.get("start")), end_second: parseDuration(form.get("end")), enabled: true,
      };
      runAndRefresh(() => api.saveRoutine(deviceID, form.get("routine_id"), routine), "Rotina salva; aguardando sincronização.");
    });
    document.querySelectorAll("[data-delete-routine]").forEach((button) => button.addEventListener("click", () => {
      if (!window.confirm("Excluir esta rotina?")) return;
      runAndRefresh(() => api.deleteRoutine(deviceID, button.dataset.deleteRoutine), "Rotina excluída.");
    }));
    document.getElementById("password-form").addEventListener("submit", (event) => {
      event.preventDefault();
      const form = new FormData(event.currentTarget);
      runAndRefresh(() => api.updatePassword(deviceID, form.get("password"), form.get("password_confirmation")), "Senha local alterada; aguardando sincronização.");
    });
    document.getElementById("rename-form").addEventListener("submit", (event) => {
      event.preventDefault();
      runAndRefresh(() => api.renameDevice(deviceID, new FormData(event.currentTarget).get("name")), "Nome atualizado.");
    });
    document.getElementById("delete-device").addEventListener("click", async () => {
      if (!window.confirm("Excluir este dispositivo e todo o histórico?")) return;
      try { await api.deleteDevice(deviceID); navigate("/"); } catch (error) { handleError(error); }
    });
  };

  const updateStatusElements = () => {
    if (!currentDevice) return;
    const status = currentDevice.status;
    const remaining = document.getElementById("remaining-time");
    const used = document.getElementById("used-time");
    const quota = document.getElementById("today-quota");
    const connection = document.getElementById("connection-status");
    if (remaining) remaining.textContent = formatDuration(status.remaining_seconds, true);
    if (used) used.textContent = formatDuration(status.used_seconds, true);
    if (quota) quota.textContent = formatDuration(status.today_quota_seconds);
    if (connection) {
      connection.textContent = connectionLabel(status);
      connection.className = `status ${status.online ? "online" : "offline"}`;
    }
  };

  const startLiveUpdates = () => {
    stopLiveUpdates();
    counterTimer = window.setInterval(() => {
      if (currentDevice && currentDevice.status.counting && currentDevice.status.remaining_seconds > 0) {
        currentDevice.status.remaining_seconds -= 1;
        currentDevice.status.used_seconds += 1;
        updateStatusElements();
      }
    }, 1000);
    synchronizationTimer = window.setInterval(async () => {
      if (!currentDevice) return;
      try {
        const incomingStatus = await api.loadStatus(currentDevice.device.id);
        currentDevice.status = window.CompassoAPI.mergeLiveStatus(currentDevice.status, incomingStatus);
        updateStatusElements();
      } catch (_error) { /* mantém contagem local */ }
    }, 2000);
  };

  const loadDevice = async (deviceID) => {
    stopLiveUpdates();
    if (!currentDevice || currentDevice.device.id !== deviceID) issuedCredential = null;
    showTopbar(true);
    try { renderDevice(await api.loadDevice(deviceID)); } catch (error) { handleError(error); }
  };

  const route = () => {
    if (currentSession && currentSession.setup_required) {
      renderInitialSetup();
      return;
    }
    if (!currentSession || !currentSession.authenticated) {
      renderLogin();
      return;
    }
    const match = /^\/devices\/([^/]+)$/.exec(window.location.pathname);
    if (match) loadDevice(decodeURIComponent(match[1]));
    else loadDevices();
  };

  logoutButton.addEventListener("click", async () => {
    try { await api.logout(); } catch (_error) { /* sessão local também será encerrada */ }
    currentSession = null;
    window.history.replaceState({}, "", "/");
    renderLogin();
  });
  window.addEventListener("popstate", route);

  const initialize = async () => {
    if (!configuredAPIBaseURL || !/^https?:\/\//.test(configuredAPIBaseURL)) {
      application.innerHTML = noticeMarkup("URL da API não configurada.", "error");
      return;
    }
    try { currentSession = await api.loadSession(); } catch (error) { handleError(error); return; }
    route();
  };

  initialize();
})();
