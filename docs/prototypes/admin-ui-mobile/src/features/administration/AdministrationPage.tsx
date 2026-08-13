import { BellRing, ChevronRight, History, KeyRound, MonitorCog, Pencil, ShieldCheck } from "lucide-react";
import { useState } from "react";
import { Link, useNavigate, useOutletContext } from "react-router-dom";
import { useAppState } from "../../app/AppState";
import { Button } from "../../components/ui/Button";
import { ConfirmDialog } from "../../components/ui/ConfirmDialog";
import { Eyebrow } from "../../components/ui/Eyebrow";
import { Surface } from "../../components/ui/Surface";
import type { Client } from "../../domain/models";
import { Page } from "../../layouts/Page";

export function AdministrationPage() {
  const { client } = useOutletContext<{ client: Client }>();
  const { getClientAdministration, getClientEvents, removeClient, showToast } = useAppState();
  const navigate = useNavigate();
  const [removeOpen, setRemoveOpen] = useState(false);
  const administration = getClientAdministration(client.id);
  const eventCount = getClientEvents(client.id).length;

  const remove = async () => {
    try {
      await removeClient(client.id);
      navigate("/");
      showToast("Cliente removido.");
      return true;
    } catch (error) {
      const detail = error instanceof Error ? error.message : "Erro desconhecido.";
      showToast(`Não foi possível remover o cliente: ${detail}`);
      return false;
    }
  };

  const rows = [
    {
      label: "Nome do cliente",
      value: client.name,
      Icon: Pencil,
      to: `/clients/${client.id}/administration/name`,
    },
    {
      label: "Pareamento do agente",
      value: administration.credentialActive === null
        ? "Status não informado"
        : administration.credentialActive ? "Ativa" : "Não gerada",
      Icon: KeyRound,
      to: `/clients/${client.id}/administration/credential`,
    },
    {
      label: "Senha local",
      value: administration.localPasswordSet ? "Configurada" : "Não configurada",
      Icon: ShieldCheck,
      to: `/clients/${client.id}/administration/password`,
    },
    {
      label: "Aviso de tempo",
      value: `${administration.warningMinutes} min antes`,
      Icon: BellRing,
      to: `/clients/${client.id}/administration/warning`,
    },
    {
      label: "Histórico",
      value: eventCount === 1 ? "1 evento" : `${eventCount} eventos`,
      Icon: History,
      to: `/clients/${client.id}/administration/history`,
    },
  ];

  return (
    <Page narrow>
      <Surface className="p-5 md:p-6">
        <Eyebrow>Configuração técnica</Eyebrow>
        <div className="flex items-start gap-3">
          <span className="grid size-11 shrink-0 place-items-center rounded-xl bg-brand-soft text-brand"><MonitorCog aria-hidden="true" size={22} /></span>
          <div>
            <h1 className="text-xl font-extrabold">Administração</h1>
            <p className="mt-1 text-sm leading-relaxed text-muted">Identidade, acesso e credenciais de {client.name}.</p>
          </div>
        </div>

        <div className="mt-6 divide-y divide-line">
          {rows.map(({ label, value, Icon, to }) => (
            <Link className="grid min-h-[4.75rem] w-full grid-cols-[2.5rem_minmax(0,1fr)_auto] items-center gap-3 py-3 text-left transition-colors hover:text-brand active:scale-[.99]" key={label} to={to}>
              <span className="grid size-10 place-items-center rounded-xl bg-surface-subtle text-brand"><Icon aria-hidden="true" size={19} /></span>
              <span><strong className="block text-sm text-ink">{label}</strong><span className="mt-1 block text-xs text-muted">{value}</span></span>
              <ChevronRight aria-hidden="true" className="text-muted" size={18} />
            </Link>
          ))}
        </div>

        <div className="mt-6 border-t border-line pt-5">
          <h2 className="text-sm font-extrabold text-danger">Zona de atenção</h2>
          <p className="mt-1 text-xs leading-relaxed text-muted">A remoção apaga limites, rotinas e o vínculo deste cliente com o Compasso.</p>
          <Button className="mt-3" fullWidth onClick={() => setRemoveOpen(true)} variant="danger">Remover cliente</Button>
        </div>
      </Surface>
      <ConfirmDialog
        confirmLabel="Remover"
        description={`“${client.name}” e todas as suas configurações serão removidos do Compasso.`}
        onClose={() => setRemoveOpen(false)}
        onConfirm={remove}
        open={removeOpen}
        title="Remover este cliente?"
      />
    </Page>
  );
}
