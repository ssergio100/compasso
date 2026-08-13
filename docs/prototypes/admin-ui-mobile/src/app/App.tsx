import { Navigate, Route, Routes } from "react-router-dom";
import { Toast } from "../components/ui/Toast";
import { AdministrationPage } from "../features/administration/AdministrationPage";
import { ClientCredentialPage } from "../features/administration/ClientCredentialPage";
import { ClientNamePage } from "../features/administration/ClientNamePage";
import { ClientPasswordPage } from "../features/administration/ClientPasswordPage";
import { ClientHistoryPage } from "../features/administration/ClientHistoryPage";
import { ClientWarningPage } from "../features/administration/ClientWarningPage";
import { AuthPage } from "../features/auth/AuthPage";
import { ProtectedRoute } from "../features/auth/ProtectedRoute";
import { ClientsPage } from "../features/clients/ClientsPage";
import { ClientFormPage } from "../features/clients/ClientFormPage";
import { UsagePage } from "../features/current-usage/UsagePage";
import { LimitsPage } from "../features/limits/LimitsPage";
import { RoutineFormPage } from "../features/routines/RoutineFormPage";
import { RoutinesPage } from "../features/routines/RoutinesPage";
import { ClientLayout } from "../layouts/ClientLayout";

export function App() {
  return (
    <>
      <a className="fixed top-2 left-2 z-50 inline-flex min-h-11 -translate-y-20 items-center rounded-lg bg-brand-dark px-3 py-2 text-sm font-bold text-white focus:translate-y-0" href="#main-content">Pular para o conteúdo</a>
      <Routes>
        <Route element={<AuthPage mode="login" />} path="/login" />
        <Route element={<AuthPage mode="setup" />} path="/setup" />
        <Route element={<ProtectedRoute />}>
          <Route element={<ClientsPage />} path="/" />
          <Route element={<ClientFormPage />} path="/clients/new" />
          <Route element={<ClientLayout />} path="/clients/:clientId">
            <Route element={<Navigate replace to="now" />} index />
            <Route element={<UsagePage />} path="now" />
            <Route element={<LimitsPage />} path="limits" />
            <Route element={<RoutinesPage />} path="routines" />
            <Route element={<RoutineFormPage />} path="routines/new" />
            <Route element={<RoutineFormPage />} path="routines/:routineId/edit" />
            <Route element={<AdministrationPage />} path="administration" />
            <Route element={<ClientNamePage />} path="administration/name" />
            <Route element={<ClientCredentialPage />} path="administration/credential" />
            <Route element={<ClientPasswordPage />} path="administration/password" />
            <Route element={<ClientWarningPage />} path="administration/warning" />
            <Route element={<ClientHistoryPage />} path="administration/history" />
          </Route>
        </Route>
        <Route element={<Navigate replace to="/" />} path="*" />
      </Routes>
      <Toast />
    </>
  );
}
