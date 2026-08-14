import { ChevronRight, LogOut, Plus } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { useAppState } from "../../app/AppState";
import { Button } from "../../components/ui/Button";
import { Eyebrow } from "../../components/ui/Eyebrow";
import { Surface } from "../../components/ui/Surface";
import { IconButton } from "../../components/ui/IconButton";
import { PageState } from "../../components/ui/PageState";
import { useAuthState } from "../auth/AuthState";
import { formatMinutes } from "../../lib/formatters";
import { AppHeader } from "../../layouts/AppHeader";
import { Page } from "../../layouts/Page";

export function ClientsPage() {
  const navigate = useNavigate();
  const { clients, dataError, dataStatus, reloadData } = useAppState();
  const { logout } = useAuthState();

  return (
    <div className="min-h-dvh">
      <AppHeader>
        <div className="flex items-center gap-2.5 text-lg font-extrabold tracking-[-.02em]">
          <span className="grid size-9 place-items-center rounded-xl bg-accent text-brand-dark">C</span>
          <span>Compasso</span>
        </div>
        <IconButton label="Sair do Compasso" onClick={async () => { await logout(); navigate("/login"); }} tone="light">
          <LogOut aria-hidden="true" size={19} />
        </IconButton>
      </AppHeader>
      <Page>
        {dataStatus === "loading" ? (
          <PageState description="Buscando clientes e configurações no servidor." kind="loading" title="Carregando clientes" />
        ) : dataStatus === "error" ? (
          <PageState actionLabel="Tentar novamente" description={dataError?.description ?? "Não foi possível carregar os clientes."} kind="error" onAction={() => void reloadData()} title={dataError?.title ?? "Erro ao carregar"} />
        ) : clients.length === 0 ? (
          <PageState actionLabel="Adicionar cliente" description="Cadastre um computador para começar a configurar limites e acompanhar o uso." kind="empty" onAction={() => navigate("/clients/new")} title="Nenhum cliente cadastrado" />
        ) : <Surface className="p-5 md:p-6">
          <Eyebrow>Visão geral</Eyebrow>
          <div className="mb-5">
            <h1 className="text-2xl font-extrabold tracking-[-.035em]">Clientes</h1>
            <p className="mt-2 max-w-[42rem] text-base leading-relaxed text-muted">Escolha quem você quer acompanhar ou configurar.</p>
          </div>
          <div className="grid gap-3 md:grid-cols-2">
            {clients.map((client) => (
              <button
                className="grid min-h-[6.75rem] w-full grid-cols-[3.25rem_minmax(0,1fr)_auto] items-center gap-3 rounded-[1.125rem] border border-line bg-surface p-4 text-left shadow-raised transition-[transform,border-color] duration-150 hover:-translate-y-px hover:border-brand/45 active:scale-[.99]"
                key={client.id}
                onClick={() => navigate(`/clients/${client.id}/now`)}
                type="button"
              >
                <span className="grid size-[3.25rem] place-items-center rounded-[1.05rem] bg-brand-soft text-lg font-extrabold text-brand">{client.initials}</span>
                <span className="min-w-0">
                  <strong className="block text-base leading-tight">{client.name}</strong>
                  <span className="mt-1 block text-sm leading-snug text-muted">{formatMinutes(client.remainingMinutes)} disponíveis hoje</span>
                </span>
                <span className="grid justify-items-end gap-2">
                  <span className="inline-flex items-center gap-1.5 text-xs font-bold text-muted">
                    <span className={`size-2 rounded-full ${client.agentOnline ? "bg-emerald-500 ring-4 ring-emerald-50" : "bg-neutral-400"}`} />
                    {client.agentOnline ? "Online" : "Offline"}
                  </span>
                  <ChevronRight aria-hidden="true" className="text-muted" size={18} />
                </span>
              </button>
            ))}
          </div>
          <Button className="mt-4" fullWidth leadingIcon={<Plus aria-hidden="true" size={18} />} onClick={() => navigate("/clients/new")} variant="dashed">
            Adicionar cliente
          </Button>
        </Surface>}
      </Page>
    </div>
  );
}
