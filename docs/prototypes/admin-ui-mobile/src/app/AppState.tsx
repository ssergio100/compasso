import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import {
  initialAuditEvents,
  initialClientAdministration,
  initialClients,
  initialLimits,
  initialRoutines,
} from "../data/mockData";
import type {
  Client,
  ClientAdministrationState,
  ClientAdministrationStates,
  ClientControlState,
  ClientControlStates,
  ClientDraft,
  DataLoadError,
  ClientOperation,
  ClientOperations,
  AuditEvent,
  Routine,
  RoutineDraft,
  WeeklyLimits,
} from "../domain/models";
import { compassoAPI, CompassoAPIError, remoteMode } from "../services/compassoApi";
import { APIContractError, mapDeviceDetail } from "../services/remoteMappers";
import { useAuthState } from "../features/auth/AuthState";

interface AppStateValue {
  dataStatus: "loading" | "ready" | "error";
  dataError: DataLoadError | null;
  clients: Client[];
  limits: WeeklyLimits;
  routines: Routine[];
  operations: ClientOperations;
  toast: string | null;
  addClient: (draft: ClientDraft) => Promise<Client>;
  renameClient: (clientId: string, name: string) => Promise<void>;
  removeClient: (clientId: string) => Promise<void>;
  getClient: (clientId: string) => Client | undefined;
  getClientAdministration: (clientId: string) => ClientAdministrationState;
  getClientOperation: (clientId: string) => ClientOperation | undefined;
  getClientEvents: (clientId: string) => AuditEvent[];
  issueClientCredential: (clientId: string) => Promise<string>;
  revokeClientCredential: (clientId: string) => Promise<void>;
  setClientLocalPassword: (clientId: string, password: string, confirmation: string) => Promise<void>;
  getClientControl: (clientId: string) => ClientControlState;
  getRoutines: (clientId: string) => Routine[];
  addBonusTime: (clientId: string, minutes: number) => Promise<boolean>;
  toggleClientPaused: (clientId: string) => Promise<boolean>;
  toggleClientBlocked: (clientId: string) => Promise<boolean>;
  retryClientOperation: (clientId: string) => void;
  saveLimits: (clientId: string, values: number[]) => Promise<boolean>;
  setClientWarningMinutes: (clientId: string, minutes: number) => Promise<boolean>;
  addRoutine: (clientId: string, draft: RoutineDraft) => Promise<void>;
  updateRoutine: (clientId: string, routineId: string, draft: RoutineDraft) => Promise<void>;
  toggleRoutineEnabled: (clientId: string, routineId: string) => Promise<void>;
  removeRoutine: (clientId: string, routineId: string) => Promise<void>;
  showToast: (message: string) => void;
  dismissToast: () => void;
  reloadData: () => Promise<void>;
}

const AppStateContext = createContext<AppStateValue | null>(null);

const defaultClientControl: ClientControlState = {
  paused: false,
  blocked: false,
  bonusMinutes: 0,
};

function clientInitials(name: string) {
  const words = name.split(/\s+/).filter(Boolean);
  return words.length > 1
    ? `${words[0][0]}${words[words.length - 1][0]}`.toUpperCase()
    : name.slice(0, 2).toUpperCase();
}

function withoutKey<T>(record: Record<string, T>, key: string) {
  return Object.fromEntries(Object.entries(record).filter(([entryKey]) => entryKey !== key));
}

function generateCredential() {
  const bytes = new Uint8Array(32);
  crypto.getRandomValues(bytes);
  const binary = Array.from(bytes, (byte) => String.fromCharCode(byte)).join("");
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

function clockToSeconds(value: string) {
  const [hours, minutes] = value.split(":").map(Number);
  return (hours * 60 + minutes) * 60;
}

function routinePayload(draft: RoutineDraft, enabled: boolean) {
  return {
    name: draft.name.trim(),
    days: Array.from({ length: 7 }, (_, day) => draft.days.includes(day)) as [boolean, boolean, boolean, boolean, boolean, boolean, boolean],
    start_second: clockToSeconds(draft.start),
    end_second: clockToSeconds(draft.end),
    enabled,
  };
}

function operationErrorDetail(error: unknown) {
  return error instanceof Error ? error.message : "Não foi possível enviar o comando.";
}

function interfaceID() {
  return typeof crypto.randomUUID === "function"
    ? crypto.randomUUID()
    : `${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`;
}

const wait = (milliseconds: number) => new Promise((resolve) => window.setTimeout(resolve, milliseconds));

export function AppStateProvider({ children }: { children: ReactNode }) {
  const { authenticated } = useAuthState();
  const operationTimers = useRef<number[]>([]);
  const dataRequestSequence = useRef(0);
  const [clients, setClients] = useState<Client[]>(remoteMode ? [] : initialClients);
  const [limits, setLimits] = useState<WeeklyLimits>(remoteMode ? {} : initialLimits);
  const [routines, setRoutines] = useState<Routine[]>(remoteMode ? [] : initialRoutines);
  const [clientControls, setClientControls] = useState<ClientControlStates>(remoteMode ? {} : Object.fromEntries(initialClients.map((client) => [client.id, { ...defaultClientControl }])));
  const [clientAdministration, setClientAdministration] = useState<ClientAdministrationStates>(remoteMode ? {} : initialClientAdministration);
  const [operations, setOperations] = useState<ClientOperations>({});
  const [auditEvents, setAuditEvents] = useState<AuditEvent[]>(remoteMode ? [] : initialAuditEvents);
  const [dataStatus, setDataStatus] = useState<"loading" | "ready" | "error">(remoteMode ? "loading" : "ready");
  const [dataError, setDataError] = useState<DataLoadError | null>(null);
  const [toast, setToast] = useState<string | null>(null);

  useEffect(() => () => operationTimers.current.forEach((timer) => window.clearTimeout(timer)), []);

  const loadRemoteData = useCallback(async (showLoading: boolean) => {
    if (!remoteMode || !authenticated) return;
    const requestSequence = ++dataRequestSequence.current;
    if (showLoading) {
      setDataStatus("loading");
      setDataError(null);
    }
    try {
      const response = await compassoAPI.listDevices();
      const details = await Promise.all(response.devices.map((device) => compassoAPI.loadDevice(device.id)));
      const mapped = details.map(mapDeviceDetail);
      if (requestSequence !== dataRequestSequence.current) return;
      const nextClients = mapped.map((item) => item.client);
      setClients(nextClients);
      setLimits(Object.fromEntries(mapped.map((item) => [item.client.id, item.limits])));
      setRoutines(mapped.flatMap((item) => item.routines));
      setClientControls(Object.fromEntries(mapped.map((item) => [item.client.id, item.control])));
      setClientAdministration(Object.fromEntries(mapped.map((item) => [item.client.id, item.administration])));
      setAuditEvents(mapped.flatMap((item) => item.events));
      setOperations((current) => Object.fromEntries(Object.entries(current).map(([clientId, operation]) => {
        const client = nextClients.find((item) => item.id === clientId);
        const confirmed = client && (operation.confirmation === "policy"
          ? client.appliedPolicyRevision >= client.policyRevision
            : operation.confirmation === "bonus" ? false
              : operation.confirmation === "pause" ? client.controlStatus === "paused"
            : operation.confirmation === "block" ? client.controlStatus === "blocked"
              : operation.confirmation === "resume" || operation.confirmation === "unblock"
                ? client.controlStatus === "active"
                : false);
        return [clientId, operation.status === "pending" && client?.agentOnline && confirmed
          ? { ...operation, status: "applied", detail: "Alteração confirmada pelo agente." }
          : operation];
      })));
      setDataError(null);
      setDataStatus("ready");
    } catch (error) {
	  if (showLoading && requestSequence === dataRequestSequence.current) {
		setDataError(error instanceof APIContractError
		  ? { title: "Versão incompatível", description: "A interface foi atualizada, mas a API instalada ainda usa um contrato antigo. Atualize o servidor Compasso." }
		  : error instanceof CompassoAPIError && error.status > 0
			? { title: "Erro da API", description: error.message }
			: { title: "Servidor indisponível", description: "Não foi possível conectar à API do Compasso. Verifique a rede e o serviço." });
        setDataStatus("error");
      }
    }
  }, [authenticated]);

  const reloadData = useCallback(() => loadRemoteData(true), [loadRemoteData]);

  useEffect(() => {
    if (!remoteMode || !authenticated) return;
    void loadRemoteData(true);
    const timer = window.setInterval(() => void loadRemoteData(false), 5000);
    return () => window.clearInterval(timer);
  }, [authenticated, loadRemoteData]);

  const getClient = useCallback(
    (clientId: string) => clients.find((client) => client.id === clientId),
    [clients],
  );

  const recordAuditEvent = useCallback((clientId: string, title: string, detail: string) => {
    setAuditEvents((current) => [{
      id: interfaceID(),
      clientId,
      title,
      detail,
      createdAt: "Agora mesmo",
    }, ...current]);
  }, []);

  const startClientOperation = useCallback((clientId: string, label: string, detail: string,
    confirmation: ClientOperation["confirmation"] = "policy") => {
    const online = clients.some((client) => client.id === clientId && client.agentOnline);
    const id = interfaceID();
    setOperations((current) => ({
      ...current,
      [clientId]: {
        id,
        clientId,
        label,
		confirmation,
        status: "pending",
        detail: online
          ? "Comando enviado. Aguardando confirmação do agente."
          : "Salvo no Compasso. Será aplicado quando o agente se conectar.",
      },
    }));
    if (!online) return id;

    if (remoteMode) return id;
    const timer = window.setTimeout(() => {
      setOperations((current) => current[clientId]?.id === id ? {
        ...current,
        [clientId]: { ...current[clientId], status: "applied", detail },
      } : current);
    }, 900);
    operationTimers.current.push(timer);
    return true;
  }, [clients]);

  const failClientOperation = useCallback((clientId: string, error: unknown) => {
    setOperations((current) => current[clientId] ? {
      ...current,
      [clientId]: { ...current[clientId], status: "error", detail: operationErrorDetail(error) },
    } : current);
  }, []);

  const addClient = useCallback(async (draft: ClientDraft) => {
    const name = draft.name.trim();
    if (remoteMode) {
      const created = await compassoAPI.createDevice(name);
      const mapped = mapDeviceDetail(await compassoAPI.loadDevice(created.id));
      setClients((current) => [...current, mapped.client]);
      setLimits((current) => ({ ...current, [created.id]: mapped.limits }));
      setRoutines((current) => [...current, ...mapped.routines]);
      setClientControls((current) => ({ ...current, [created.id]: mapped.control }));
      setClientAdministration((current) => ({ ...current, [created.id]: mapped.administration }));
      setAuditEvents((current) => [...mapped.events, ...current]);
      return mapped.client;
    }
    const initials = clientInitials(name);
    const id = interfaceID();

    const client: Client = {
      id,
      name,
      initials,
      agentOnline: false,
      graphicalSessionActive: false,
      monitoringActive: false,
      countingTime: false,
      policyRevision: 1,
      appliedPolicyRevision: 0,
		controlRevision: 1,
		appliedControlRevision: 0,
		actualState: "offline",
		controlStatus: "offline",
      remainingMinutes: 0,
      usedMinutes: 0,
      dailyLimitMinutes: 0,
      lastSynchronization: "Ainda não sincronizado",
    };

    setClients((current) => [...current, client]);
    setLimits((current) => ({ ...current, [id]: [0, 0, 0, 0, 0, 0, 0] }));
    setClientControls((current) => ({ ...current, [id]: { ...defaultClientControl } }));
    setClientAdministration((current) => ({
      ...current,
      [id]: { credentialActive: false, localPasswordSet: false, warningMinutes: 10 },
    }));
    return client;
  }, []);

  const renameClient = useCallback(async (clientId: string, name: string) => {
    const normalizedName = name.trim();
    if (remoteMode) await compassoAPI.renameDevice(clientId, normalizedName);
    setClients((current) => current.map((client) => (
      client.id === clientId
        ? { ...client, name: normalizedName, initials: clientInitials(normalizedName) }
        : client
    )));
    if (remoteMode) await reloadData();
    else recordAuditEvent(clientId, "Cliente renomeado", `O nome foi alterado para “${normalizedName}”.`);
  }, [recordAuditEvent, reloadData]);

  const removeClient = useCallback(async (clientId: string) => {
    if (remoteMode) {
      dataRequestSequence.current += 1;
      await compassoAPI.deleteDevice(clientId);
      await reloadData();
      return;
    }
    setClients((current) => current.filter((client) => client.id !== clientId));
    setLimits((current) => withoutKey(current, clientId));
    setRoutines((current) => current.filter((routine) => routine.clientId !== clientId));
    setClientControls((current) => withoutKey(current, clientId));
    setClientAdministration((current) => withoutKey(current, clientId));
    setOperations((current) => withoutKey(current, clientId));
    setAuditEvents((current) => current.filter((event) => event.clientId !== clientId));
  }, [reloadData]);

  const getClientAdministration = useCallback(
    (clientId: string) => clientAdministration[clientId] ?? {
      credentialActive: false,
      localPasswordSet: false,
      warningMinutes: 10,
    },
    [clientAdministration],
  );

  const getClientOperation = useCallback(
    (clientId: string) => operations[clientId],
    [operations],
  );

  const getClientEvents = useCallback(
    (clientId: string) => auditEvents.filter((event) => event.clientId === clientId),
    [auditEvents],
  );

  const issueClientCredential = useCallback(async (clientId: string) => {
    const credential = remoteMode ? (await compassoAPI.issueToken(clientId)).device_token : generateCredential();
    setClientAdministration((current) => ({
      ...current,
      [clientId]: { ...current[clientId], credentialActive: true },
    }));
    if (remoteMode) await reloadData();
    else recordAuditEvent(clientId, "Credencial gerada", "Uma nova credencial de pareamento foi emitida.");
    return credential;
  }, [recordAuditEvent, reloadData]);

  const revokeClientCredential = useCallback(async (clientId: string) => {
    if (remoteMode) await compassoAPI.revokeToken(clientId);
    setClientAdministration((current) => ({
      ...current,
      [clientId]: { ...current[clientId], credentialActive: false },
    }));
    if (remoteMode) await reloadData();
    else recordAuditEvent(clientId, "Credencial revogada", "O agente perdeu acesso à credencial anterior.");
  }, [recordAuditEvent, reloadData]);

  const setClientLocalPassword = useCallback(async (clientId: string, password: string, confirmation: string) => {
    if (remoteMode) await compassoAPI.updatePassword(clientId, password, confirmation);
    setClientAdministration((current) => ({
      ...current,
      [clientId]: { ...current[clientId], localPasswordSet: true },
    }));
    if (remoteMode) await reloadData();
    else recordAuditEvent(clientId, "Senha local atualizada", "Uma nova senha local foi definida.");
  }, [recordAuditEvent, reloadData]);

  const getRoutines = useCallback(
    (clientId: string) => routines.filter((routine) => routine.clientId === clientId),
    [routines],
  );

  const getClientControl = useCallback(
    (clientId: string) => clientControls[clientId] ?? defaultClientControl,
    [clientControls],
  );

  const updateClientControl = useCallback(
    (clientId: string, update: (current: ClientControlState) => ClientControlState) => {
      setClientControls((current) => ({
        ...current,
        [clientId]: update(current[clientId] ?? defaultClientControl),
      }));
    },
    [],
  );

  const addBonusTime = useCallback(async (clientId: string, minutes: number) => {
    const accepted = startClientOperation(clientId, "Adicionar tempo", "Tempo adicional confirmado pelo agente.", "bonus");
    if (!accepted) return false;
    try {
      if (remoteMode) {
        const confirmation = await compassoAPI.addBonus(clientId, minutes);
        void (async () => {
          const deadline = Date.now() + 2 * 60 * 1000;
          while (Date.now() < deadline) {
            const status = await compassoAPI.loadBonusStatus(clientId, confirmation.operation_id);
            if (status.acknowledged) {
              await reloadData();
              setOperations((current) => current[clientId]?.id === accepted ? {
                ...current,
                [clientId]: { ...current[clientId], status: "applied", detail: "Tempo adicional confirmado pelo agente." },
              } : current);
              return;
            }
            await wait(2000);
          }
        })().catch((error) => failClientOperation(clientId, error));
        return true;
      }
      updateClientControl(clientId, (current) => ({ ...current, bonusMinutes: current.bonusMinutes + minutes }));
      recordAuditEvent(clientId, "Tempo adicional", `${minutes} minutos adicionados ao limite de hoje.`);
      return true;
    } catch (error) {
      failClientOperation(clientId, error);
      return false;
    }
  }, [failClientOperation, recordAuditEvent, reloadData, startClientOperation, updateClientControl]);

  const toggleClientPaused = useCallback(async (clientId: string) => {
    const paused = clientControls[clientId]?.paused ?? false;
    const accepted = startClientOperation(clientId, paused ? "Retomar uso" : "Pausar uso", paused ? "Uso retomado pelo agente." : "Uso pausado pelo agente.", paused ? "resume" : "pause");
    if (!accepted) return false;
    updateClientControl(clientId, (current) => ({ ...current, paused: !paused }));
    try {
      if (remoteMode) await compassoAPI.queueCommand(clientId, paused ? "resume_monitoring" : "pause_monitoring");
      if (remoteMode) await reloadData();
      else {
        setClients((current) => current.map((client) => client.id === clientId
          ? { ...client, controlStatus: paused ? "active" : "paused" }
          : client));
        recordAuditEvent(clientId, paused ? "Uso retomado" : "Uso pausado", paused ? "A contagem foi retomada." : "A contagem foi pausada manualmente.");
      }
      return true;
    } catch (error) {
      updateClientControl(clientId, (current) => ({ ...current, paused }));
      failClientOperation(clientId, error);
      return false;
    }
  }, [clientControls, failClientOperation, recordAuditEvent, reloadData, startClientOperation, updateClientControl]);

  const toggleClientBlocked = useCallback(async (clientId: string) => {
    const blocked = clientControls[clientId]?.blocked ?? false;
    const accepted = startClientOperation(clientId, blocked ? "Remover bloqueio" : "Bloquear agora", blocked ? "Bloqueio removido pelo agente." : "Bloqueio aplicado pelo agente.", blocked ? "unblock" : "block");
    if (!accepted) return false;
    updateClientControl(clientId, (current) => ({ ...current, blocked: !blocked }));
    try {
      if (remoteMode) await compassoAPI.queueCommand(clientId, blocked ? "clear_manual_block" : "block_now");
      if (remoteMode) await reloadData();
      else {
        setClients((current) => current.map((client) => client.id === clientId
          ? { ...client, controlStatus: blocked ? "active" : "blocked", actualState: blocked ? "unblocked" : "blocked" }
          : client));
        recordAuditEvent(clientId, blocked ? "Bloqueio removido" : "Uso bloqueado", blocked ? "O acesso ao computador foi liberado." : "O acesso foi bloqueado manualmente.");
      }
      return true;
    } catch (error) {
      updateClientControl(clientId, (current) => ({ ...current, blocked }));
      failClientOperation(clientId, error);
      return false;
    }
  }, [clientControls, failClientOperation, recordAuditEvent, reloadData, startClientOperation, updateClientControl]);

  const retryClientOperation = useCallback((clientId: string) => {
    const operation = operations[clientId];
    if (operation) startClientOperation(clientId, operation.label, "Comando confirmado pelo agente.", operation.confirmation);
  }, [operations, startClientOperation]);

  const saveLimits = useCallback(async (clientId: string, values: number[]) => {
    const accepted = startClientOperation(clientId, "Atualizar limites", "Novos limites aplicados pelo agente.");
    if (!accepted) return false;
    try {
      if (remoteMode) await compassoAPI.updatePolicy(clientId, values.map((minutes) => minutes * 60), clientAdministration[clientId]?.warningMinutes ?? 10);
      setLimits((current) => ({ ...current, [clientId]: [...values] }));
      if (remoteMode) await reloadData();
      else recordAuditEvent(clientId, "Limites atualizados", "A regra semanal de tempo foi alterada.");
      return true;
    } catch (error) {
      failClientOperation(clientId, error);
      return false;
    }
  }, [clientAdministration, failClientOperation, recordAuditEvent, reloadData, startClientOperation]);

  const setClientWarningMinutes = useCallback(async (clientId: string, minutes: number) => {
    const accepted = startClientOperation(clientId, "Atualizar aviso", "Antecedência do aviso aplicada pelo agente.");
    if (!accepted) return false;
    try {
      if (remoteMode) await compassoAPI.updatePolicy(clientId, (limits[clientId] ?? Array(7).fill(0)).map((value) => value * 60), minutes);
      setClientAdministration((current) => ({
        ...current,
        [clientId]: { ...current[clientId], warningMinutes: minutes },
      }));
      if (remoteMode) await reloadData();
      else recordAuditEvent(clientId, "Aviso atualizado", `O aviso será exibido ${minutes} minutos antes do fim do tempo.`);
      return true;
    } catch (error) {
      failClientOperation(clientId, error);
      return false;
    }
  }, [failClientOperation, limits, recordAuditEvent, reloadData, startClientOperation]);

  const addRoutine = useCallback(async (clientId: string, draft: RoutineDraft) => {
    if (remoteMode) {
      await compassoAPI.saveRoutine(clientId, undefined, routinePayload(draft, true));
      await reloadData();
      return;
    }
    setRoutines((current) => [
      ...current,
      { ...draft, clientId, enabled: true, id: `routine-${interfaceID()}` },
    ]);
    recordAuditEvent(clientId, "Rotina criada", `A rotina “${draft.name}” foi adicionada.`);
  }, [recordAuditEvent, reloadData]);

  const updateRoutine = useCallback(async (clientId: string, routineId: string, draft: RoutineDraft) => {
    const routine = routines.find((item) => item.clientId === clientId && item.id === routineId);
    if (remoteMode) {
      await compassoAPI.saveRoutine(clientId, routineId, routinePayload(draft, routine?.enabled ?? true));
      await reloadData();
      return;
    }
    setRoutines((current) => current.map((routine) => (
      routine.clientId === clientId && routine.id === routineId
        ? { ...routine, ...draft }
        : routine
    )));
    recordAuditEvent(clientId, "Rotina atualizada", `A rotina “${draft.name}” foi alterada.`);
  }, [recordAuditEvent, reloadData, routines]);

  const toggleRoutineEnabled = useCallback(async (clientId: string, routineId: string) => {
    const routine = routines.find((item) => item.clientId === clientId && item.id === routineId);
    if (!routine) return;
    if (remoteMode) {
      await compassoAPI.saveRoutine(clientId, routineId, routinePayload(routine, !routine.enabled));
      await reloadData();
      return;
    }
    setRoutines((current) => current.map((routine) => (
      routine.clientId === clientId && routine.id === routineId
        ? { ...routine, enabled: !routine.enabled }
        : routine
    )));
    recordAuditEvent(clientId, routine.enabled ? "Rotina desativada" : "Rotina ativada", `A rotina “${routine.name}” foi ${routine.enabled ? "desativada" : "ativada"}.`);
  }, [recordAuditEvent, reloadData, routines]);

  const removeRoutine = useCallback(async (clientId: string, routineId: string) => {
    const routine = routines.find((item) => item.clientId === clientId && item.id === routineId);
    if (remoteMode) {
      await compassoAPI.deleteRoutine(clientId, routineId);
      await reloadData();
      return;
    }
    setRoutines((current) => current.filter((routine) => (
      routine.clientId !== clientId || routine.id !== routineId
    )));
    if (routine) recordAuditEvent(clientId, "Rotina removida", `A rotina “${routine.name}” foi excluída.`);
  }, [recordAuditEvent, reloadData, routines]);

  const showToast = useCallback((message: string) => {
    setToast(message);
    const timer = window.setTimeout(() => setToast(null), 2400);
    operationTimers.current.push(timer);
  }, []);

  const dismissToast = useCallback(() => setToast(null), []);

  const value = useMemo<AppStateValue>(
    () => ({
      dataStatus,
      dataError,
      clients,
      limits,
      routines,
      operations,
      toast,
      addClient,
      renameClient,
      removeClient,
      getClient,
      getClientAdministration,
      getClientOperation,
      getClientEvents,
      issueClientCredential,
      revokeClientCredential,
      setClientLocalPassword,
      getClientControl,
      getRoutines,
      addBonusTime,
      toggleClientPaused,
      toggleClientBlocked,
      retryClientOperation,
      saveLimits,
      setClientWarningMinutes,
      addRoutine,
      updateRoutine,
      toggleRoutineEnabled,
      removeRoutine,
      showToast,
      dismissToast,
      reloadData,
    }),
    [
      addBonusTime,
      addClient,
      addRoutine,
      getClientAdministration,
      getClientEvents,
      getClientOperation,
      dismissToast,
      getClient,
      getClientControl,
      getRoutines,
      clients,
      dataError,
      dataStatus,
      limits,
      operations,
      issueClientCredential,
      removeClient,
      reloadData,
      renameClient,
      revokeClientCredential,
      routines,
      removeRoutine,
      showToast,
      setClientLocalPassword,
      setClientWarningMinutes,
      toast,
      toggleClientBlocked,
      toggleClientPaused,
      toggleRoutineEnabled,
      retryClientOperation,
      saveLimits,
      updateRoutine,
    ],
  );

  return <AppStateContext.Provider value={value}>{children}</AppStateContext.Provider>;
}

export function useAppState() {
  const context = useContext(AppStateContext);
  if (!context) throw new Error("useAppState must be used inside AppStateProvider");
  return context;
}
