import { BellRing, Save } from "lucide-react";
import { useEffect, useState } from "react";
import { useOutletContext } from "react-router-dom";
import { useAppState } from "../../app/AppState";
import { Button } from "../../components/ui/Button";
import { OperationStatus } from "../../components/ui/OperationStatus";
import { Surface } from "../../components/ui/Surface";
import type { Client } from "../../domain/models";
import { Page } from "../../layouts/Page";
import { AdministrationSubpageHeader } from "./AdministrationSubpageHeader";

const options = [5, 10, 15, 30];

export function ClientWarningPage() {
  const { client } = useOutletContext<{ client: Client }>();
  const { getClientAdministration, getClientOperation, retryClientOperation, setClientWarningMinutes, showToast } = useAppState();
  const administration = getClientAdministration(client.id);
  const [minutes, setMinutes] = useState(administration.warningMinutes);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    setMinutes(administration.warningMinutes);
  }, [administration.warningMinutes]);
  const operation = getClientOperation(client.id);
  const changed = minutes !== administration.warningMinutes;

  const save = async () => {
    setSaving(true);
    const saved = await setClientWarningMinutes(client.id, minutes);
    setSaving(false);
    showToast(saved
      ? (client.agentOnline ? "Aviso salvo e enviado ao agente." : "Aviso salvo. Aguardando o agente conectar.")
      : "Não foi possível salvar o aviso.");
  };

  return (
    <Page narrow>
      <Surface className="p-5 md:p-6">
        <AdministrationSubpageHeader clientId={client.id} description="Defina quando o usuário será avisado antes do tempo disponível terminar." title="Aviso de tempo" />
        <div className="flex items-start gap-3 rounded-2xl bg-surface-subtle p-4">
          <span className="grid size-10 shrink-0 place-items-center rounded-xl bg-brand-soft text-brand"><BellRing aria-hidden="true" size={20} /></span>
          <p className="text-sm leading-relaxed text-muted">O aviso é exibido pelo agente no próprio computador.</p>
        </div>

        <fieldset className="mt-5">
          <legend className="text-sm font-extrabold">Avisar com antecedência</legend>
          <div className="mt-3 grid grid-cols-2 gap-2.5 sm:grid-cols-4">
            {options.map((option) => (
              <button
                aria-pressed={minutes === option}
                className={`min-h-12 rounded-xl border px-3 text-sm font-bold transition-colors ${minutes === option ? "border-brand bg-brand-soft text-brand" : "border-line bg-surface text-muted hover:border-brand/45"}`}
                key={option}
                onClick={() => setMinutes(option)}
                type="button"
              >{option} min</button>
            ))}
          </div>
        </fieldset>

        {changed ? <p className="mt-4 text-center text-xs font-bold text-warning" role="status">Alteração ainda não salva</p> : null}
        <div className="mt-4 grid grid-cols-2 gap-2.5">
          <Button disabled={!changed || saving} onClick={() => setMinutes(administration.warningMinutes)} variant="secondary">Cancelar</Button>
          <Button disabled={!changed || saving} leadingIcon={<Save aria-hidden="true" size={17} />} onClick={save}>{saving ? "Salvando…" : "Salvar aviso"}</Button>
        </div>
        <OperationStatus operation={operation?.label === "Atualizar aviso" ? operation : undefined} onRetry={() => retryClientOperation(client.id)} />
      </Surface>
    </Page>
  );
}
