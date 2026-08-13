import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { compassoAPI, remoteMode } from "../../services/compassoApi";

interface AuthStateValue {
  authenticated: boolean;
  adminConfigured: boolean;
  checking: boolean;
  error: string | null;
  setupRequired: boolean;
  login: (login: string, password: string) => Promise<boolean>;
  logout: () => Promise<void>;
  setupAdmin: (login: string, password: string) => Promise<boolean>;
}

const AuthStateContext = createContext<AuthStateValue | null>(null);

export function AuthStateProvider({ children, initiallyAuthenticated = !remoteMode }: { children: ReactNode; initiallyAuthenticated?: boolean }) {
  const [authenticated, setAuthenticated] = useState(initiallyAuthenticated);
  const [adminConfigured, setAdminConfigured] = useState(true);
  const [checking, setChecking] = useState(remoteMode);
  const [error, setError] = useState<string | null>(null);
  const [setupRequired, setSetupRequired] = useState(false);

  useEffect(() => {
    if (!remoteMode) return;
    compassoAPI.loadSession()
      .then((session) => {
        setAuthenticated(session.authenticated);
        setSetupRequired(session.setup_required);
        setAdminConfigured(!session.setup_required);
      })
      .catch(() => {
        setAuthenticated(false);
        setSetupRequired(true);
        setAdminConfigured(false);
        setError("Não foi possível verificar a sessão no servidor.");
      })
      .finally(() => setChecking(false));
  }, []);

  const value = useMemo<AuthStateValue>(() => ({
    authenticated,
    adminConfigured,
    checking,
    error,
    setupRequired,
    login: async (login, password) => {
      setError(null);
      if (!remoteMode) {
        const valid = login.trim().length > 0 && password.length >= 6;
        if (valid) setAuthenticated(true);
        return valid;
      }
      try {
        const session = await compassoAPI.login(login, password);
        setAuthenticated(session.authenticated);
        setSetupRequired(session.setup_required);
        setAdminConfigured(!session.setup_required);
        return session.authenticated;
      } catch {
        setAuthenticated(false);
        setError("Usuário ou senha inválidos.");
        return false;
      }
    },
    logout: async () => {
      try {
        if (remoteMode) await compassoAPI.logout();
      } finally {
        setAuthenticated(false);
      }
    },
    setupAdmin: async (login, password) => {
      setError(null);
      try {
        if (remoteMode) {
          const session = await compassoAPI.setup(login, password, password);
          setAdminConfigured(!session.setup_required);
          setSetupRequired(session.setup_required);
          setAuthenticated(session.authenticated);
          return session.authenticated;
        }
        setAdminConfigured(true);
        setSetupRequired(false);
        setAuthenticated(true);
        return true;
      } catch {
        setAuthenticated(false);
        setError("Não foi possível concluir a configuração inicial.");
        return false;
      }
    },
  }), [adminConfigured, authenticated, checking, error, setupRequired]);

  return <AuthStateContext.Provider value={value}>{children}</AuthStateContext.Provider>;
}

export function useAuthState() {
  const context = useContext(AuthStateContext);
  if (!context) throw new Error("useAuthState must be used inside AuthStateProvider");
  return context;
}
