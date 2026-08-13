import { Copy, KeyRound, RefreshCw, ShieldOff } from "lucide-react";
import { useRef, useState } from "react";
import { useOutletContext } from "react-router-dom";
import { useAppState } from "../../app/AppState";
import { Button } from "../../components/ui/Button";
import { ConfirmDialog } from "../../components/ui/ConfirmDialog";
import { Surface } from "../../components/ui/Surface";
import type { Client } from "../../domain/models";
import { copyText } from "../../lib/clipboard";
import { Page } from "../../layouts/Page";
import { AdministrationSubpageHeader } from "./AdministrationSubpageHeader";

type Confirmation = "replace" | "revoke" | null;

export function ClientCredentialPage() {
  const { client } = useOutletContext<{ client: Client }>();
  const {
    getClientAdministration,
    issueClientCredential,
    revokeClientCredential,
    showToast,
  } = useAppState();
  const administration = getClientAdministration(client.id);
  const [issuedCredential, setIssuedCredential] = useState<string | null>(null);
  const [confirmation, setConfirmation] = useState<Confirmation>(null);
  const actionInFlight = useRef(false);
  const credentialKnown = administration.credentialActive !== null;

  const issue = async () => {
    if (actionInFlight.current) return;
    actionInFlight.current = true;
    try {
      setIssuedCredential(await issueClientCredential(client.id));
      showToast("Nova credencial gerada.");
      return true;
    } catch {
      showToast("Não foi possível gerar a credencial.");
      return false;
    } finally {
      actionInFlight.current = false;
    }
  };

  const revoke = async () => {
    if (actionInFlight.current) return;
    actionInFlight.current = true;
    try {
      await revokeClientCredential(client.id);
      setIssuedCredential(null);
      showToast("Credencial revogada.");
      return true;
    } catch {
      showToast("Não foi possível revogar a credencial.");
      return false;
    } finally {
      actionInFlight.current = false;
    }
  };

  const copyPairingValue = async (value: string, label: string) => {
    const copied = await copyText(value);
    showToast(copied ? `${label} copiado.` : "Não foi possível copiar automaticamente.");
  };

  return (
    <Page narrow>
      <Surface className="p-5 md:p-6">
        <AdministrationSubpageHeader
          clientId={client.id}
          description="Dados usados para conectar o agente instalado neste computador ao Compasso."
          title="Pareamento do agente"
        />

        <div className="flex items-center gap-3 rounded-2xl bg-surface-subtle p-4">
          <span className="grid size-10 shrink-0 place-items-center rounded-xl bg-brand-soft text-brand"><KeyRound aria-hidden="true" size={20} /></span>
          <div>
            <strong className="block text-sm">{!credentialKnown ? "Status da credencial indisponível" : administration.credentialActive ? "Credencial ativa" : "Sem credencial ativa"}</strong>
            <p className="mt-1 text-xs leading-relaxed text-muted">
              {!credentialKnown
                ? "O servidor atual não informa se existe uma credencial ativa. Gerar uma nova invalida a anterior."
                : administration.credentialActive ? "O agente pode usar a credencial atual para se autenticar." : "Gere uma credencial para iniciar o pareamento."}
            </p>
          </div>
        </div>

        <section className="mt-5 rounded-2xl border border-line bg-surface-subtle p-4" aria-labelledby="pairing-data-title">
          <h2 className="text-sm font-extrabold" id="pairing-data-title">Dados de pareamento</h2>
          <p className="mt-1 text-xs leading-relaxed text-muted">Informe estes valores na configuração do agente. O identificador não é secreto.</p>

          <div className="mt-4">
            <span className="block text-xs font-bold text-muted">Identificador do dispositivo <span className="font-mono font-normal">(device_id)</span></span>
            <div className="mt-2 grid gap-2 sm:grid-cols-[minmax(0,1fr)_auto]">
              <code className="min-w-0 break-all rounded-xl bg-surface px-3 py-3 font-mono text-xs leading-relaxed text-ink">{client.id}</code>
              <Button
                aria-label="Copiar identificador do dispositivo"
                leadingIcon={<Copy aria-hidden="true" size={17} />}
                onClick={() => copyPairingValue(client.id, "Identificador")}
                variant="secondary"
              >Copiar</Button>
            </div>
          </div>

          {issuedCredential ? (
            <div className="mt-4 border-t border-line pt-4" role="status">
              <strong className="block text-sm text-brand">Nova credencial — copie agora</strong>
              <p className="mt-1 text-xs leading-relaxed text-muted">Por segurança, o conteúdo do <span className="font-mono">device_token</span> será exibido somente desta vez.</p>
              <span className="mt-3 block text-xs font-bold text-muted">Credencial secreta <span className="font-mono font-normal">(device_token)</span></span>
              <div className="mt-2 grid gap-2 sm:grid-cols-[minmax(0,1fr)_auto]">
                <code className="min-w-0 break-all rounded-xl bg-surface px-3 py-3 font-mono text-xs leading-relaxed text-ink">{issuedCredential}</code>
                <Button
                  aria-label="Copiar credencial secreta"
                  leadingIcon={<Copy aria-hidden="true" size={17} />}
                  onClick={() => copyPairingValue(issuedCredential, "Credencial")}
                  variant="secondary"
                >Copiar</Button>
              </div>
            </div>
          ) : (
            <p className="mt-4 border-t border-line pt-4 text-xs leading-relaxed text-muted">
              {!credentialKnown
                ? "A credencial não pode ser consultada. Gere uma nova somente se precisar configurar o agente novamente."
                : administration.credentialActive
                ? "A credencial ativa não pode ser consultada. Gere uma nova somente se precisar configurar o agente novamente."
                : "Ainda não há uma credencial. Gere uma para concluir o pareamento."}
            </p>
          )}
        </section>

        <div className="mt-6 grid gap-2.5">
          {administration.credentialActive || !credentialKnown ? (
            <>
              <Button fullWidth leadingIcon={<RefreshCw aria-hidden="true" size={17} />} onClick={() => setConfirmation("replace")} variant="secondary">Gerar nova credencial</Button>
              <Button fullWidth leadingIcon={<ShieldOff aria-hidden="true" size={17} />} onClick={() => setConfirmation("revoke")} variant="danger">Revogar credencial</Button>
            </>
          ) : (
            <Button fullWidth leadingIcon={<KeyRound aria-hidden="true" size={17} />} onClick={issue}>Gerar credencial</Button>
          )}
        </div>
      </Surface>
      <ConfirmDialog
        confirmLabel={confirmation === "replace" ? "Substituir" : "Revogar"}
        description={confirmation === "replace"
          ? "A credencial atual deixará de funcionar imediatamente e uma nova será exibida uma única vez."
          : "O agente perderá o acesso e permanecerá desconectado até que uma nova credencial seja configurada."}
        onClose={() => setConfirmation(null)}
        onConfirm={confirmation === "replace" ? issue : revoke}
        open={confirmation !== null}
        title={confirmation === "replace" ? "Substituir a credencial?" : "Revogar a credencial?"}
      />
    </Page>
  );
}
