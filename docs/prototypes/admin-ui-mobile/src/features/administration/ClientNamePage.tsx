import { Save } from "lucide-react";
import { useState, type FormEvent } from "react";
import { useNavigate, useOutletContext } from "react-router-dom";
import { useAppState } from "../../app/AppState";
import { Button } from "../../components/ui/Button";
import { Field } from "../../components/ui/Field";
import { Surface } from "../../components/ui/Surface";
import type { Client } from "../../domain/models";
import { Page } from "../../layouts/Page";
import { AdministrationSubpageHeader } from "./AdministrationSubpageHeader";

export function ClientNamePage() {
  const { client } = useOutletContext<{ client: Client }>();
  const { renameClient, showToast } = useAppState();
  const navigate = useNavigate();
  const [name, setName] = useState(client.name);
  const [touched, setTouched] = useState(false);
  const normalizedName = name.trim();
  const error = touched && !normalizedName ? "Informe um nome para o cliente." : undefined;
  const changed = normalizedName !== client.name;

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setTouched(true);
    if (!normalizedName) return;

    try {
      await renameClient(client.id, normalizedName);
      navigate(`/clients/${client.id}/administration`);
      showToast("Nome atualizado.");
    } catch {
      showToast("Não foi possível atualizar o nome.");
    }
  }

  return (
    <Page narrow>
      <Surface className="p-5 md:p-6">
        <AdministrationSubpageHeader
          clientId={client.id}
          description="Este nome identifica o computador em toda a administração."
          title="Nome do cliente"
        />
        <form noValidate onSubmit={save}>
          <Field
            autoComplete="off"
            autoFocus
            error={error}
            help="Use um nome curto e fácil de reconhecer."
            id="client-edit-name"
            label="Nome"
            maxLength={80}
            onBlur={() => setTouched(true)}
            onChange={(event) => setName(event.currentTarget.value)}
            required
            value={name}
          />
          <Button className="mt-6" disabled={!changed || !normalizedName} fullWidth leadingIcon={<Save aria-hidden="true" size={17} />} type="submit">Salvar nome</Button>
        </form>
      </Surface>
    </Page>
  );
}
