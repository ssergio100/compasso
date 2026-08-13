import { ArrowLeft } from "lucide-react";
import { Outlet, useNavigate, useParams } from "react-router-dom";
import { useAppState } from "../app/AppState";
import { IconButton } from "../components/ui/IconButton";
import { PageState } from "../components/ui/PageState";
import { AppHeader } from "./AppHeader";
import { ClientNavigation } from "./ClientNavigation";
import { Page } from "./Page";

export function ClientLayout() {
  const navigate = useNavigate();
  const { clientId } = useParams();
  const { getClient } = useAppState();
  const client = clientId ? getClient(clientId) : undefined;

  if (!client) return (
    <div className="min-h-dvh">
      <AppHeader><strong className="text-lg">Compasso</strong></AppHeader>
      <Page narrow>
        <PageState actionLabel="Voltar para clientes" description="Este cliente não existe mais ou os dados não puderam ser carregados." kind="error" onAction={() => navigate("/")} title="Cliente indisponível" />
      </Page>
    </div>
  );

  return (
    <div className="min-h-dvh pb-24">
      <AppHeader>
        <div className="flex min-w-0 items-center gap-3">
          <IconButton label="Voltar para clientes" onClick={() => navigate("/")} tone="light">
            <ArrowLeft aria-hidden="true" size={20} />
          </IconButton>
          <span className="grid size-11 shrink-0 place-items-center rounded-[.9rem] bg-white/14 text-lg font-extrabold">{client.initials}</span>
          <span className="min-w-0">
            <strong className="line-clamp-2 block text-base leading-tight">{client.name}</strong>
            <small className="mt-1 block text-sm text-white/76">{client.agentOnline ? "Agente conectado" : "Agente desconectado"}</small>
          </span>
        </div>
      </AppHeader>
      <Outlet context={{ client }} />
      <ClientNavigation />
    </div>
  );
}
