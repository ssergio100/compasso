import { Clock3, History } from "lucide-react";
import { useOutletContext } from "react-router-dom";
import { useAppState } from "../../app/AppState";
import { PageState } from "../../components/ui/PageState";
import { Surface } from "../../components/ui/Surface";
import type { Client } from "../../domain/models";
import { Page } from "../../layouts/Page";
import { AdministrationSubpageHeader } from "./AdministrationSubpageHeader";

export function ClientHistoryPage() {
  const { client } = useOutletContext<{ client: Client }>();
  const { getClientEvents } = useAppState();
  const events = getClientEvents(client.id);

  return (
    <Page narrow>
      <Surface className="p-5 md:p-6">
        <AdministrationSubpageHeader clientId={client.id} description="Acompanhe alterações administrativas e comunicações importantes do agente." title="Histórico" />
        {events.length === 0 ? (
          <PageState description="As alterações e sincronizações deste cliente aparecerão aqui." headingLevel="h2" kind="empty" title="Nenhum evento registrado" />
        ) : (
          <ol className="divide-y divide-line" aria-label="Eventos do cliente">
            {events.map((event) => (
              <li className="grid grid-cols-[2.5rem_minmax(0,1fr)] gap-3 py-4" key={event.id}>
                <span className="grid size-10 place-items-center rounded-xl bg-surface-subtle text-brand">
                  <History aria-hidden="true" size={18} />
                </span>
                <div className="min-w-0">
                  <strong className="block text-sm">{event.title}</strong>
                  <p className="mt-1 text-xs leading-relaxed text-muted">{event.detail}</p>
                  <time className="mt-2 inline-flex items-center gap-1.5 text-xs font-bold text-muted"><Clock3 aria-hidden="true" size={14} />{event.createdAt}</time>
                </div>
              </li>
            ))}
          </ol>
        )}
      </Surface>
    </Page>
  );
}
