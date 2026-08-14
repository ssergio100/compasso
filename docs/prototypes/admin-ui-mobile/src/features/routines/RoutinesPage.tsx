import { CalendarPlus, ChevronRight, Plus } from "lucide-react";
import { Link, useOutletContext } from "react-router-dom";
import { useAppState } from "../../app/AppState";
import { RoutineIcon } from "../../components/RoutineIcon";
import { Eyebrow } from "../../components/ui/Eyebrow";
import { Surface } from "../../components/ui/Surface";
import type { Client, Routine } from "../../domain/models";
import { formatRoutineDays } from "../../lib/formatters";
import { Page } from "../../layouts/Page";
import { findRoutineConflict } from "./routineConflicts";

export function RoutinesPage() {
  const { client } = useOutletContext<{ client: Client }>();
  const { getRoutines, showToast, toggleRoutineEnabled } = useAppState();
  const routines = getRoutines(client.id);
  const enabledCount = routines.filter((routine) => routine.enabled).length;

  async function toggleRoutine(routine: Routine) {
    if (!routine.enabled) {
      const conflict = findRoutineConflict(routine, routines, routine.id);
      if (conflict) {
        showToast(`Não foi possível ativar: há conflito com “${conflict.name}”.`);
        return;
      }
    }

    try {
      await toggleRoutineEnabled(client.id, routine.id);
      showToast(routine.enabled ? "Rotina desativada." : "Rotina ativada.");
    } catch {
      showToast("Não foi possível alterar a rotina.");
    }
  }

  return (
    <Page narrow>
      <Surface className="p-5 md:p-6">
        <Eyebrow>Agenda semanal</Eyebrow>
        <h1 className="text-xl font-extrabold">Rotinas</h1>
        <p className="mt-2 text-sm leading-relaxed text-muted">Bloqueios que acontecem automaticamente nos horários escolhidos.</p>

        {routines.length > 0 ? (
          <>
            <div className="mt-6 mb-3 flex items-end justify-between gap-3">
              <h2 className="text-base font-extrabold">Cadastradas</h2>
              <span className="text-xs font-bold text-muted">{enabledCount} {enabledCount === 1 ? "ativa" : "ativas"}</span>
            </div>
            <div className="divide-y divide-line rounded-2xl border border-line px-3 sm:px-4">
              {routines.map((routine) => (
                <article className="grid grid-cols-[minmax(0,1fr)_3rem] items-center gap-1" key={routine.id}>
                  <Link
                    aria-label={`Editar rotina ${routine.name}`}
                    className="grid min-h-[5rem] min-w-0 grid-cols-[2.875rem_minmax(0,1fr)_auto] items-center gap-3 py-3 pr-2 transition-colors hover:text-brand active:scale-[.99]"
                    to={`/clients/${client.id}/routines/${routine.id}/edit`}
                  >
                    <RoutineIcon muted={!routine.enabled} name={routine.icon} />
                    <span className="min-w-0">
                      <span className="flex flex-wrap items-center gap-2">
                        <strong className="block text-sm text-ink">{routine.name}</strong>
                        {!routine.enabled && <span className="rounded-full bg-line px-2 py-0.5 text-[.65rem] font-extrabold text-muted">INATIVA</span>}
                      </span>
                      <span className="mt-1 block text-xs leading-relaxed text-muted">
                        {formatRoutineDays(routine.days)} · {routine.start} até {routine.end}{routine.end < routine.start ? " do dia seguinte" : routine.end === routine.start ? " · dia inteiro" : ""}
                      </span>
                    </span>
                    <ChevronRight aria-hidden="true" className="text-muted" size={18} />
                  </Link>
                  <button
                    aria-checked={routine.enabled}
                    aria-label={`${routine.enabled ? "Desativar" : "Ativar"} rotina ${routine.name}`}
                    className="grid size-11 place-items-center rounded-xl active:scale-[.97]"
                    onClick={() => toggleRoutine(routine)}
                    role="switch"
                    type="button"
                  >
                    <span className={`relative h-6 w-10 rounded-full transition-colors ${routine.enabled ? "bg-brand" : "bg-line-strong"}`}>
                      <span className={`absolute top-1 left-1 size-4 rounded-full bg-white shadow-raised transition-transform ${routine.enabled ? "translate-x-4" : "translate-x-0"}`} />
                    </span>
                  </button>
                </article>
              ))}
            </div>
          </>
        ) : (
          <div className="mt-6 grid justify-items-center rounded-2xl bg-surface-subtle px-5 py-8 text-center">
            <span className="grid size-12 place-items-center rounded-2xl bg-brand-soft text-brand"><CalendarPlus aria-hidden="true" size={24} /></span>
            <h2 className="mt-4 text-base font-extrabold">Nenhuma rotina criada</h2>
            <p className="mt-1 max-w-[26rem] text-sm leading-relaxed text-muted">Crie um bloqueio recorrente para horários como estudo, refeições ou hora de dormir.</p>
          </div>
        )}

        <Link className="mt-4 inline-flex min-h-11 w-full items-center justify-center gap-2 rounded-xl border border-dashed border-line-strong px-4 py-2.5 text-sm font-bold text-brand transition-[transform,background-color,border-color] hover:border-brand hover:bg-brand-soft/45 active:scale-[.97]" to={`/clients/${client.id}/routines/new`}>
          <Plus aria-hidden="true" size={18} /> Criar rotina
        </Link>
      </Surface>
    </Page>
  );
}
