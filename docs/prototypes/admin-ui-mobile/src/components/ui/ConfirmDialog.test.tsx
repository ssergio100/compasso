import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { vi } from "vitest";
import { ConfirmDialog } from "./ConfirmDialog";

it("blocks duplicate confirmations while an asynchronous action is running", async () => {
  const user = userEvent.setup();
  let finish!: () => void;
  const pending = new Promise<void>((resolve) => { finish = resolve; });
  const onConfirm = vi.fn(() => pending);

  render(
    <ConfirmDialog
      confirmLabel="Substituir"
      description="Gera uma credencial nova."
      onClose={() => undefined}
      onConfirm={onConfirm}
      open
      title="Substituir credencial?"
    />,
  );

  await user.dblClick(screen.getByRole("button", { name: "Substituir" }));

  expect(onConfirm).toHaveBeenCalledTimes(1);
  expect(screen.getByRole("button", { name: "Processando…" })).toBeDisabled();
  await act(async () => finish());
});
