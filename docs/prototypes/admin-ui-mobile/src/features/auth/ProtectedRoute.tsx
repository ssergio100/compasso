import { Navigate, Outlet, useLocation } from "react-router-dom";
import { PageState } from "../../components/ui/PageState";
import { useAuthState } from "./AuthState";

export function ProtectedRoute() {
  const { authenticated, checking } = useAuthState();
  const location = useLocation();
  if (checking) return <div className="p-4"><PageState description="Verificando sua sessão administrativa." kind="loading" title="Carregando Compasso" /></div>;
  return authenticated ? <Outlet /> : <Navigate replace state={{ from: location.pathname }} to="/login" />;
}
