import { ArrowLeft, Monitor, Plus } from "lucide-react";
import { useRef, useState, type FormEvent } from "react";
import { useNavigate } from "react-router-dom";
import { useAppState } from "../../app/AppState";
import { Button } from "../../components/ui/Button";
import { Eyebrow } from "../../components/ui/Eyebrow";
import { Field } from "../../components/ui/Field";
import { IconButton } from "../../components/ui/IconButton";
import { Surface } from "../../components/ui/Surface";
import { AppHeader } from "../../layouts/AppHeader";
import { Page } from "../../layouts/Page";

export function ClientFormPage() {
  const { addClient, showToast } = useAppState();
  const navigate = useNavigate();
  const nameRef = useRef<HTMLInputElement>(null);
  const [name, setName] = useState("");
  const [nameTouched, setNameTouched] = useState(false);
  const normalizedName = name.trim();
  const nameError = nameTouched && normalizedName.length === 0 ? "Informe um nome para o cliente." : undefined;

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!normalizedName) {
      setNameTouched(true);
      nameRef.current?.focus();
      return;
    }

    try {
      const client = await addClient({ name: normalizedName });
      navigate(`/clients/${client.id}/now`);
      showToast("Cliente adicionado. Agora configure os limites e o pareamento.");
    } catch {
      showToast("Não foi possível adicionar o cliente.");
    }
  }

  return (
    <div className="min-h-dvh">
      <AppHeader>
        <div className="flex min-w-0 items-center gap-3">
          <IconButton label="Cancelar novo cliente" onClick={() => navigate("/")} tone="light">
            <ArrowLeft aria-hidden="true" size={20} />
          </IconButton>
          <div>
            <strong className="block text-base leading-tight">Novo cliente</strong>
            <small className="mt-1 block text-sm text-white/76">Computador administrado</small>
          </div>
        </div>
      </AppHeader>

      <Page narrow>
        <Surface className="p-5 md:p-6">
          <Eyebrow>Identificação</Eyebrow>
          <h1 className="text-2xl font-extrabold tracking-[-.035em]">Adicionar cliente</h1>
          <p className="mt-2 max-w-[34rem] text-sm leading-relaxed text-muted">
            Escolha um nome fácil de reconhecer. A conexão com o computador será configurada depois do cadastro.
          </p>

          <form className="mt-6" noValidate onSubmit={submit}>
            <Field
              autoComplete="off"
              autoFocus
              error={nameError}
              help="Use o nome do computador ou do local onde ele fica."
              id="client-name"
              label="Nome do cliente"
              maxLength={80}
              onBlur={() => setNameTouched(true)}
              onChange={(event) => {
                setName(event.currentTarget.value);
                if (event.currentTarget.value.trim()) setNameTouched(false);
              }}
              placeholder="Ex.: Computador da sala"
              ref={nameRef}
              required
              value={name}
            />

            <div className="mt-5 flex items-start gap-3 rounded-2xl bg-surface-subtle p-4">
              <span className="grid size-10 shrink-0 place-items-center rounded-xl bg-brand-soft text-brand">
                <Monitor aria-hidden="true" size={20} />
              </span>
              <div>
                <strong className="block text-sm">O cliente começará offline</strong>
                <p className="mt-1 text-xs leading-relaxed text-muted">
                  Após adicionar, configure os limites e gere a credencial usada pelo agente para se conectar.
                </p>
              </div>
            </div>

            <div className="mt-6 grid grid-cols-2 gap-2.5">
              <Button onClick={() => navigate("/")} type="button" variant="secondary">Cancelar</Button>
              <Button disabled={!normalizedName} leadingIcon={<Plus aria-hidden="true" size={18} />} type="submit">Adicionar</Button>
            </div>
          </form>
        </Surface>
      </Page>
    </div>
  );
}
