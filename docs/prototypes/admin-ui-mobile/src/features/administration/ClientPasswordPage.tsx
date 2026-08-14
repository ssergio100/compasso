import { Save, ShieldCheck } from "lucide-react";
import { useState, type FormEvent } from "react";
import { useNavigate, useOutletContext } from "react-router-dom";
import { useAppState } from "../../app/AppState";
import { Button } from "../../components/ui/Button";
import { Field } from "../../components/ui/Field";
import { Surface } from "../../components/ui/Surface";
import type { Client } from "../../domain/models";
import { Page } from "../../layouts/Page";
import { AdministrationSubpageHeader } from "./AdministrationSubpageHeader";

export function ClientPasswordPage() {
  const { client } = useOutletContext<{ client: Client }>();
  const { getClientAdministration, setClientLocalPassword, showToast } = useAppState();
  const navigate = useNavigate();
  const administration = getClientAdministration(client.id);
  const [password, setPassword] = useState("");
  const [confirmation, setConfirmation] = useState("");
  const [passwordTouched, setPasswordTouched] = useState(false);
  const [confirmationTouched, setConfirmationTouched] = useState(false);
  const passwordError = passwordTouched && !password ? "Informe a nova senha." : undefined;
  const confirmationError = confirmationTouched && confirmation !== password ? "As senhas não coincidem." : undefined;
  const canSave = password.length > 0 && confirmation === password;

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPasswordTouched(true);
    setConfirmationTouched(true);
    if (!canSave) return;

    try {
      await setClientLocalPassword(client.id, password, confirmation);
      navigate(`/clients/${client.id}/administration`);
      showToast("Senha local atualizada.");
    } catch {
      showToast("Não foi possível atualizar a senha local.");
    }
  }

  return (
    <Page narrow>
      <Surface className="p-5 md:p-6">
        <AdministrationSubpageHeader
          clientId={client.id}
          description="Usada pelo responsável para autorizar tempo adicional diretamente no computador."
          title={administration.localPasswordSet ? "Alterar senha local" : "Configurar senha local"}
        />

        <div className="mb-5 flex items-start gap-3 rounded-2xl bg-surface-subtle p-4">
          <ShieldCheck aria-hidden="true" className="mt-0.5 shrink-0 text-brand" size={20} />
          <p className="text-xs leading-relaxed text-muted">A senha não será exibida depois de salva. A alteração será aplicada ao agente na próxima sincronização.</p>
        </div>

        <form noValidate onSubmit={save}>
          <Field
            autoComplete="new-password"
            autoFocus
            error={passwordError}
            id="local-password"
            label="Nova senha"
            onBlur={() => setPasswordTouched(true)}
            onChange={(event) => setPassword(event.currentTarget.value)}
            required
            type="password"
            value={password}
          />
          <div className="mt-4">
            <Field
              autoComplete="new-password"
              error={confirmationError}
              id="local-password-confirmation"
              label="Confirmar senha"
              onBlur={() => setConfirmationTouched(true)}
              onChange={(event) => setConfirmation(event.currentTarget.value)}
              required
              type="password"
              value={confirmation}
            />
          </div>
          <Button className="mt-6" disabled={!canSave} fullWidth leadingIcon={<Save aria-hidden="true" size={17} />} type="submit">Salvar senha</Button>
        </form>
      </Surface>
    </Page>
  );
}
