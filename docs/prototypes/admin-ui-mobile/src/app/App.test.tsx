import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { ESTUDOS_CLIENT_ID, QUARTO_CLIENT_ID } from "../data/mockData";
import { App } from "./App";
import { AppStateProvider } from "./AppState";
import { AuthStateProvider } from "../features/auth/AuthState";

function renderApp(path = "/") {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <AuthStateProvider>
        <AppStateProvider>
          <App />
        </AppStateProvider>
      </AuthStateProvider>
    </MemoryRouter>,
  );
}

describe("Compasso administration routes", () => {
	it("handles controls when randomUUID is unavailable over local HTTP", async () => {
		const original = crypto.randomUUID;
		Object.defineProperty(crypto, "randomUUID", { configurable: true, value: undefined });
		try {
			renderApp(`/clients/${QUARTO_CLIENT_ID}/now`);
			await userEvent.click(screen.getByRole("button", { name: "Pausar" }));
			expect(screen.getByRole("button", { name: "Retomar" })).toBeVisible();
		} finally {
			Object.defineProperty(crypto, "randomUUID", { configurable: true, value: original });
		}
	});
  it("opens a client from the client list", async () => {
    const user = userEvent.setup();
    renderApp();

    await user.click(screen.getByRole("button", { name: /Computador do quarto/ }));

    expect(screen.getByRole("heading", { name: "Uso liberado" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Bloquear" }).querySelector("svg")).not.toBeNull();
    expect(screen.queryByRole("button", { name: "Abrir administração" })).not.toBeInTheDocument();
  });

  it("shows all seven daily limits without horizontal disclosure", () => {
    renderApp(`/clients/${QUARTO_CLIENT_ID}/limits`);

    expect(screen.getAllByRole("tab")).toHaveLength(7);
    expect(screen.getByRole("tab", { name: /Sáb/ })).toBeVisible();
  });

  it("keeps limit changes as a draft until they are saved", async () => {
    const user = userEvent.setup();
    renderApp(`/clients/${QUARTO_CLIENT_ID}/limits`);

    const range = screen.getByRole("slider", { name: /Tempo disponível/ });
    expect(range).toHaveValue("180");
    fireEvent.change(range, { target: { value: "240" } });

    expect(screen.getByText("Alterações ainda não salvas")).toBeVisible();
    await user.click(screen.getByRole("button", { name: "Cancelar" }));
    expect(range).toHaveValue("180");

    fireEvent.change(range, { target: { value: "240" } });
    await user.click(screen.getByRole("button", { name: "Salvar limites" }));
    expect(screen.getByText(/Atualizar limites · Aguardando sincronização/)).toBeVisible();
  });

  it("creates a client from the add-client form", async () => {
    const user = userEvent.setup();
    renderApp();

    await user.click(screen.getByRole("button", { name: "Adicionar cliente" }));

    expect(screen.getByRole("heading", { name: "Adicionar cliente" })).toBeVisible();
    expect(screen.getByRole("button", { name: "Adicionar" })).toBeDisabled();

    await user.type(screen.getByLabelText("Nome do cliente"), "Computador da sala");
    await user.click(screen.getByRole("button", { name: "Adicionar" }));

    expect(await screen.findByText("Computador da sala")).toBeVisible();
    expect(screen.getByRole("heading", { name: "Agente desconectado" })).toBeVisible();

    await user.click(screen.getByRole("button", { name: "Voltar para clientes" }));

    await user.click(screen.getByRole("button", { name: /Computador da sala/ }));
    await user.click(screen.getByRole("link", { name: "Administração" }));

    expect(screen.getByText("Não gerada")).toBeVisible();
    expect(screen.getByText("Não configurada")).toBeVisible();
  });

  it("adds time from the clock and updates today's available time", async () => {
    const user = userEvent.setup();
    renderApp(`/clients/${QUARTO_CLIENT_ID}/now`);

    await user.click(screen.getByRole("button", { name: "Mais tempo" }));

    expect(screen.getByRole("dialog", { name: "Adicionar tempo" })).toBeVisible();
    expect(screen.getByText("+00:30")).toBeVisible();

    await user.click(screen.getByRole("button", { name: "Adicionar 0h 30min" }));

    expect(screen.getByText("02:12")).toBeVisible();
    expect(screen.getByText("+0h 30min")).toBeVisible();
  });

  it("gives paused and blocked controls an explicit visual state", async () => {
    const user = userEvent.setup();
    renderApp(`/clients/${QUARTO_CLIENT_ID}/now`);

    await user.click(screen.getByRole("button", { name: "Pausar" }));

    expect(screen.getByRole("heading", { name: "Uso pausado" })).toBeVisible();
    expect(screen.getByRole("button", { name: "Retomar" })).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByText(/Pausar uso · Aguardando sincronização/)).toBeVisible();

    await user.click(screen.getByRole("button", { name: "Bloquear" }));

    expect(screen.getByRole("heading", { name: "Uso bloqueado" })).toBeVisible();
    expect(screen.getByRole("button", { name: "Desbloquear" })).toHaveAttribute("aria-pressed", "true");
  });

  it("shows the independent computer connection states", () => {
    renderApp(`/clients/${QUARTO_CLIENT_ID}/now`);

    expect(screen.getByRole("heading", { name: "Estado do computador" })).toBeVisible();
    expect(screen.getByText("Sessão gráfica")).toBeVisible();
    expect(screen.getByText("Contagem de tempo")).toBeVisible();
    expect(screen.getByText("Em andamento")).toBeVisible();
  });

  it("keeps routine creation as an explicit two-step flow", async () => {
    const user = userEvent.setup();
    renderApp(`/clients/${QUARTO_CLIENT_ID}/routines/new`);

    await user.type(screen.getByLabelText("Nome"), "Leitura");
    await user.click(screen.getByRole("button", { name: "Continuar" }));

    expect(screen.getByText("Etapa 2 de 2 · Horários")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Salvar rotina" })).toBeEnabled();
  });

  it("shows a useful empty state when a client has no routines", () => {
    renderApp(`/clients/${ESTUDOS_CLIENT_ID}/routines`);

    expect(screen.getByRole("heading", { name: "Nenhuma rotina criada" })).toBeVisible();
    expect(screen.getByRole("link", { name: "Criar rotina" })).toBeVisible();
  });

  it("edits, disables, enables and deletes a routine", async () => {
    const user = userEvent.setup();
    renderApp(`/clients/${QUARTO_CLIENT_ID}/routines`);

    await user.click(screen.getByRole("link", { name: "Editar rotina Tempo de estudo" }));
    expect(screen.getByRole("heading", { name: "Editar rotina" })).toBeVisible();

    const name = screen.getByLabelText("Nome");
    await user.clear(name);
    await user.type(name, "Estudo focado");
    await user.click(screen.getByRole("button", { name: "Continuar" }));
    await user.click(screen.getByRole("button", { name: "Salvar alterações" }));

    const disable = await screen.findByRole("switch", { name: "Desativar rotina Estudo focado" });
    await user.click(disable);
    expect(await screen.findByRole("switch", { name: "Ativar rotina Estudo focado" })).toHaveAttribute("aria-checked", "false");

    await user.click(screen.getByRole("switch", { name: "Ativar rotina Estudo focado" }));
    expect(await screen.findByRole("switch", { name: "Desativar rotina Estudo focado" })).toHaveAttribute("aria-checked", "true");

    await user.click(screen.getByRole("link", { name: "Editar rotina Estudo focado" }));
    await user.click(screen.getByRole("button", { name: "Excluir rotina" }));
    expect(screen.getByRole("dialog", { name: "Excluir esta rotina?" })).toBeVisible();
    await user.click(screen.getByRole("button", { name: "Excluir" }));

    await waitFor(() => expect(screen.queryByRole("link", { name: "Editar rotina Estudo focado" })).not.toBeInTheDocument());
  });

  it("renames a client from administration", async () => {
    const user = userEvent.setup();
    renderApp(`/clients/${QUARTO_CLIENT_ID}/administration`);

    await user.click(screen.getByRole("link", { name: /Nome do cliente/ }));
    const name = screen.getByLabelText("Nome");
    await user.clear(name);
    await user.type(name, "Computador principal");
    await user.click(screen.getByRole("button", { name: "Salvar nome" }));

    expect(screen.getByText("Computador principal", { selector: "strong" })).toBeVisible();
    expect(screen.getByText("CP")).toBeVisible();
  });

  it("replaces and revokes an agent credential", async () => {
    const user = userEvent.setup();
    renderApp(`/clients/${QUARTO_CLIENT_ID}/administration/credential`);

    expect(screen.getByText("Credencial ativa")).toBeVisible();
    await user.click(screen.getByRole("button", { name: "Gerar nova credencial" }));
    await user.click(screen.getByRole("button", { name: "Substituir" }));

    expect(screen.getByRole("heading", { name: "Dados de pareamento" })).toBeVisible();
    expect(screen.getByText(QUARTO_CLIENT_ID)).toBeVisible();
    expect(screen.getByText("Nova credencial — copie agora")).toBeVisible();
    expect(screen.getByText(/^[A-Za-z0-9_-]{43}$/)).toBeVisible();

    await user.click(screen.getByRole("button", { name: "Revogar credencial" }));
    await user.click(screen.getByRole("button", { name: "Revogar" }));

    expect(screen.getByText("Sem credencial ativa")).toBeVisible();
    expect(screen.getByRole("button", { name: "Gerar credencial" })).toBeVisible();
  });

  it("validates and updates the local password", async () => {
    const user = userEvent.setup();
    renderApp(`/clients/${QUARTO_CLIENT_ID}/administration/password`);

    await user.type(screen.getByLabelText("Nova senha"), "segredo");
    await user.type(screen.getByLabelText("Confirmar senha"), "diferente");
    await user.tab();
    expect(screen.getByText("As senhas não coincidem.")).toBeVisible();

    await user.clear(screen.getByLabelText("Confirmar senha"));
    await user.type(screen.getByLabelText("Confirmar senha"), "segredo");
    await user.click(screen.getByRole("button", { name: "Salvar senha" }));

    expect(await screen.findByRole("heading", { name: "Administração" })).toBeVisible();
    expect(screen.getByText("Senha local atualizada.")).toBeVisible();
  });

  it("updates warning time and records it in history", async () => {
    const user = userEvent.setup();
    renderApp(`/clients/${QUARTO_CLIENT_ID}/administration/warning`);

    await user.click(screen.getByRole("button", { name: "15 min" }));
    expect(screen.getByText("Alteração ainda não salva")).toBeVisible();
    await user.click(screen.getByRole("button", { name: "Salvar aviso" }));
    expect(screen.getByText(/Atualizar aviso · Aguardando sincronização/)).toBeVisible();

    await user.click(screen.getByRole("button", { name: "Voltar para administração" }));
    await user.click(screen.getByRole("link", { name: /Histórico/ }));
    expect(screen.getByText("Aviso atualizado")).toBeVisible();
    expect(screen.getByText(/15 minutos antes/)).toBeVisible();
  });

  it("supports logout, login and expired-session feedback", async () => {
    const user = userEvent.setup();
    renderApp();

    await user.click(screen.getByRole("button", { name: "Sair do Compasso" }));
    expect(screen.getByRole("heading", { name: "Entrar no Compasso" })).toBeVisible();

    await user.type(screen.getByLabelText("Usuário"), "admin");
    await user.type(screen.getByLabelText("Senha"), "compasso");
    await user.click(screen.getByRole("button", { name: "Entrar" }));
    expect(screen.getByRole("heading", { name: "Clientes" })).toBeVisible();
  });

  it("shows a recoverable error for an unavailable client", () => {
    renderApp("/clients/cliente-inexistente/now");

    expect(screen.getByRole("alert")).toHaveTextContent("Cliente indisponível");
    expect(screen.getByRole("button", { name: "Voltar para clientes" })).toBeVisible();
  });

  it("removes a client and all navigation to it", async () => {
    const user = userEvent.setup();
    renderApp(`/clients/${QUARTO_CLIENT_ID}/administration`);

    await user.click(screen.getByRole("button", { name: "Remover cliente" }));
    expect(screen.getByRole("dialog", { name: "Remover este cliente?" })).toBeVisible();
    await user.click(screen.getByRole("button", { name: "Remover" }));

    expect(screen.getByRole("heading", { name: "Clientes" })).toBeVisible();
    expect(screen.queryByRole("button", { name: /Computador do quarto/ })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Notebook de estudos/ })).toBeVisible();
  });
});
