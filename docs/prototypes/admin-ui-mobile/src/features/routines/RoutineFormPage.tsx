import { AlertTriangle, ArrowLeft, Check, Save, Trash2 } from "lucide-react";
import { useMemo, useState } from "react";
import { Navigate, useNavigate, useOutletContext, useParams } from "react-router-dom";
import { useAppState } from "../../app/AppState";
import { RoutineIcon, routineIconOptions } from "../../components/RoutineIcon";
import { Button } from "../../components/ui/Button";
import { ConfirmDialog } from "../../components/ui/ConfirmDialog";
import { Field } from "../../components/ui/Field";
import { IconButton } from "../../components/ui/IconButton";
import { Surface } from "../../components/ui/Surface";
import type { Client, RoutineDraft } from "../../domain/models";
import { Page } from "../../layouts/Page";
import { findRoutineConflict } from "./routineConflicts";

const days = ["Dom", "Seg", "Ter", "Qua", "Qui", "Sex", "Sáb"];

const initialDraft: RoutineDraft = {
  name: "",
  icon: "moon",
  start: "08:00",
  end: "09:00",
  days: [0, 1, 2, 3, 4],
};

export function RoutineFormPage() {
  const { client } = useOutletContext<{ client: Client }>();
  const { routineId } = useParams();
  const {
    addRoutine,
    getRoutines,
    removeRoutine,
    showToast,
    updateRoutine,
  } = useAppState();
  const navigate = useNavigate();
  const routines = getRoutines(client.id);
  const routine = routineId ? routines.find((item) => item.id === routineId) : undefined;
  const isEditing = Boolean(routineId);
  const [step, setStep] = useState<1 | 2>(1);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [draft, setDraft] = useState<RoutineDraft>(() => routine ? {
    name: routine.name,
    icon: routine.icon,
    start: routine.start,
    end: routine.end,
    days: [...routine.days],
  } : initialDraft);
  const previewName = draft.name.trim() || "Nome da rotina";
  const nextDay = draft.end < draft.start;
  const fullDay = draft.end === draft.start;
  const conflict = useMemo(
    () => draft.days.length > 0 ? findRoutineConflict(draft, routines, routineId) : undefined,
    [draft, routineId, routines],
  );
  const conflictBlocksSave = Boolean(conflict && (routine?.enabled ?? true));
  const canContinue = draft.name.trim().length > 0;
  const canSave = canContinue && draft.days.length > 0 && !conflictBlocksSave;

  if (isEditing && !routine) {
    return <Navigate replace to={`/clients/${client.id}/routines`} />;
  }

  const goBack = () => {
    if (step === 2) setStep(1);
    else navigate(`/clients/${client.id}/routines`);
  };

  const toggleDay = (day: number) => {
    setDraft((current) => ({
      ...current,
      days: current.days.includes(day)
        ? current.days.filter((item) => item !== day)
        : [...current.days, day].sort(),
    }));
  };

  const save = async () => {
    if (!canSave) return;
    const normalizedDraft = { ...draft, name: draft.name.trim() };
    setSaving(true);
    try {
      if (routine) {
        await updateRoutine(client.id, routine.id, normalizedDraft);
        showToast("Rotina atualizada.");
      } else {
        await addRoutine(client.id, normalizedDraft);
        showToast("Rotina adicionada.");
      }
      navigate(`/clients/${client.id}/routines`);
    } catch {
      showToast("Não foi possível salvar a rotina.");
    } finally {
      setSaving(false);
    }
  };

  const remove = async () => {
    if (!routine) return;
    setSaving(true);
    try {
      await removeRoutine(client.id, routine.id);
      navigate(`/clients/${client.id}/routines`);
      showToast("Rotina excluída.");
    } catch {
      setDeleteOpen(false);
      showToast("Não foi possível excluir a rotina.");
    } finally {
      setSaving(false);
    }
  };

  return (
    <Page narrow>
      <Surface className="p-5 md:p-6">
        <div className="mb-6 flex items-center gap-3">
          <IconButton label={step === 1 ? `Cancelar ${isEditing ? "edição" : "nova rotina"}` : "Voltar para identificação"} onClick={goBack}>
            <ArrowLeft aria-hidden="true" size={19} />
          </IconButton>
          <div>
            <h1 className="text-lg font-extrabold">{step === 1 ? (isEditing ? "Editar rotina" : "Nova rotina") : previewName}</h1>
            <span className="mt-1 block text-xs text-muted">Etapa {step} de 2 · {step === 1 ? "Identificação" : "Horários"}</span>
          </div>
        </div>

        {step === 1 ? (
          <>
            <Field
              autoComplete="off"
              id="routine-name"
              label="Nome"
              maxLength={40}
              onChange={(event) => {
                const name = event.currentTarget.value;
                setDraft((current) => ({ ...current, name }));
              }}
              placeholder="Ex.: Hora de dormir"
              value={draft.name}
            />

            <fieldset className="mt-5 rounded-2xl border border-line bg-surface-subtle p-4">
              <legend className="px-1 text-sm font-extrabold">Ícone</legend>
              <div className="mt-2 grid grid-cols-6 gap-2 max-[420px]:grid-cols-4" role="radiogroup" aria-label="Ícone da rotina">
                {routineIconOptions.map(({ name, label, Icon }) => {
                  const selected = draft.icon === name;
                  return (
                    <button
                      aria-checked={selected}
                      aria-label={label}
                      className={`relative grid aspect-square min-h-11 place-items-center rounded-xl border bg-surface transition-colors ${selected ? "border-brand text-brand ring-2 ring-brand/20" : "border-line text-muted hover:border-brand/45 hover:text-brand"}`}
                      key={name}
                      onClick={() => setDraft((current) => ({ ...current, icon: name }))}
                      role="radio"
                      type="button"
                    >
                      <Icon aria-hidden="true" size={21} />
                      {selected ? <Check aria-hidden="true" className="absolute top-1 right-1" size={11} /> : null}
                    </button>
                  );
                })}
              </div>

            </fieldset>

            <div className="mt-5 flex items-center gap-3 rounded-2xl bg-surface-subtle p-4">
              <RoutineIcon name={draft.icon} />
              <span><strong className="block text-sm">{previewName}</strong><span className="mt-1 block text-xs text-muted">Prévia da rotina</span></span>
            </div>
            <Button className="mt-4" disabled={!canContinue} fullWidth onClick={() => setStep(2)}>Continuar</Button>

            {routine && (
              <div className="mt-6 border-t border-line pt-5">
                <h2 className="text-sm font-extrabold text-danger">Excluir rotina</h2>
                <p className="mt-1 text-xs leading-relaxed text-muted">O bloqueio automático deixará de existir em todos os dias selecionados.</p>
                <Button className="mt-3" fullWidth leadingIcon={<Trash2 aria-hidden="true" size={17} />} onClick={() => setDeleteOpen(true)} variant="danger">Excluir rotina</Button>
              </div>
            )}
          </>
        ) : (
          <>
            <div className="flex items-center gap-3 rounded-2xl bg-surface-subtle p-4">
              <RoutineIcon name={draft.icon} />
              <span><strong className="block text-sm">{previewName}</strong><span className="mt-1 block text-xs text-muted">Quando esta rotina deve bloquear o uso?</span></span>
            </div>

            <div className="mt-5 grid grid-cols-2 gap-3">
              <Field id="routine-start" label="De" onChange={(event) => {
                const start = event.currentTarget.value;
                setDraft((current) => ({ ...current, start }));
              }} type="time" value={draft.start} />
              <Field id="routine-end" label="Até" onChange={(event) => {
                const end = event.currentTarget.value;
                setDraft((current) => ({ ...current, end }));
              }} type="time" value={draft.end} />
            </div>

            <fieldset className="mt-5">
              <legend className="text-sm font-extrabold">Dias em que a rotina começa</legend>
              <div className="mt-3 grid grid-cols-4 gap-2 sm:grid-cols-7">
                {days.map((day, index) => {
                  const selected = draft.days.includes(index);
                  return (
                    <button
                      aria-pressed={selected}
                      className={`min-h-11 rounded-xl border px-2 text-xs font-bold transition-colors ${selected ? "border-brand bg-brand-soft text-brand" : "border-line bg-surface text-muted hover:border-brand/45"}`}
                      key={day}
                      onClick={() => toggleDay(index)}
                      type="button"
                    >
                      {day}
                    </button>
                  );
                })}
              </div>
            </fieldset>

            <p className="mt-4 text-xs leading-relaxed text-muted">
              {fullDay
                ? "Início e fim iguais bloqueiam o dia inteiro nos dias selecionados."
                : nextDay
                  ? `A rotina termina no dia seguinte, às ${draft.end}. O horário final não faz parte do bloqueio.`
                  : "O horário final não faz parte do bloqueio, permitindo que outra rotina comece no mesmo instante."}
            </p>

            {conflict && (
              <div className={`mt-4 flex items-start gap-3 rounded-2xl p-4 ${conflictBlocksSave ? "bg-danger-soft text-danger" : "bg-warning-soft text-warning"}`} role={conflictBlocksSave ? "alert" : "status"}>
                <AlertTriangle aria-hidden="true" className="mt-0.5 shrink-0" size={19} />
                <div>
                  <strong className="block text-sm">Conflito com “{conflict.name}”</strong>
                  <p className="mt-1 text-xs leading-relaxed">
                    {conflictBlocksSave
                      ? "Ajuste os dias ou horários antes de salvar."
                      : "Você pode salvar esta rotina inativa, mas precisará resolver o conflito antes de ativá-la."}
                  </p>
                </div>
              </div>
            )}

            {draft.days.length === 0 && <p className="mt-4 text-xs font-bold text-danger" role="alert">Selecione pelo menos um dia.</p>}
            <Button className="mt-4" disabled={!canSave || saving} fullWidth leadingIcon={<Save aria-hidden="true" size={17} />} onClick={save}>
              {saving ? "Salvando…" : isEditing ? "Salvar alterações" : "Salvar rotina"}
            </Button>
          </>
        )}
      </Surface>
      {routine && (
        <ConfirmDialog
          confirmLabel="Excluir"
          description={`A rotina “${routine.name}” será removida e não bloqueará mais o uso.`}
          onClose={() => setDeleteOpen(false)}
          onConfirm={remove}
          open={deleteOpen}
          title="Excluir esta rotina?"
        />
      )}
    </Page>
  );
}
