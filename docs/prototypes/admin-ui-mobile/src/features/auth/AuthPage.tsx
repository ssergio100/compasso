import { KeyRound, LogIn, ShieldCheck } from "lucide-react";
import { useEffect, useState, type FormEvent } from "react";
import { Link, useLocation, useNavigate, useSearchParams } from "react-router-dom";
import { Button } from "../../components/ui/Button";
import { Field } from "../../components/ui/Field";
import { Surface } from "../../components/ui/Surface";
import { AppHeader } from "../../layouts/AppHeader";
import { Page } from "../../layouts/Page";
import { useAuthState } from "./AuthState";

export function AuthPage({ mode }: { mode: "login" | "setup" }) {
  const { checking, error, login, setupAdmin, setupRequired } = useAuthState();
  const navigate = useNavigate();
  const location = useLocation();
  const [searchParams] = useSearchParams();
  const [userName, setUserName] = useState("");
  const [password, setPassword] = useState("");
  const [confirmation, setConfirmation] = useState("");
  const [submitted, setSubmitted] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const isSetup = mode === "setup";
  const passwordError = submitted && password.length < 6 ? "Use pelo menos 6 caracteres." : undefined;
  const confirmationError = submitted && isSetup && password !== confirmation ? "As senhas não coincidem." : undefined;
  const userError = submitted && !userName.trim() ? "Informe o usuário administrador." : undefined;

  useEffect(() => {
    if (!checking && setupRequired && !isSetup) navigate("/setup", { replace: true });
  }, [checking, isSetup, navigate, setupRequired]);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitted(true);
    if (!userName.trim() || password.length < 6 || (isSetup && password !== confirmation)) return;
    setSubmitting(true);
    const accepted = isSetup ? await setupAdmin(userName, password) : await login(userName, password);
    setSubmitting(false);
    if (!accepted) return;
    const destination = (location.state as { from?: string } | null)?.from ?? "/";
    navigate(destination, { replace: true });
  }

  return (
    <div className="min-h-dvh">
      <AppHeader>
        <div className="flex items-center gap-2.5 text-lg font-extrabold tracking-[-.02em]">
          <span className="grid size-9 place-items-center rounded-xl bg-accent text-brand-dark">C</span>
          <span>Compasso</span>
        </div>
      </AppHeader>
      <Page narrow>
        <Surface className="p-5 md:p-6">
          <span className="grid size-11 place-items-center rounded-xl bg-brand-soft text-brand">
            {isSetup ? <ShieldCheck aria-hidden="true" size={22} /> : <KeyRound aria-hidden="true" size={21} />}
          </span>
          <h1 className="mt-4 text-2xl font-extrabold tracking-[-.035em]">{isSetup ? "Configurar administrador" : "Entrar no Compasso"}</h1>
          <p className="mt-2 text-sm leading-relaxed text-muted">
            {isSetup ? "Crie o acesso que será usado para administrar os computadores." : "Use suas credenciais administrativas para continuar."}
          </p>

          {searchParams.get("reason") === "expired" ? (
            <p className="mt-4 rounded-xl bg-warning-soft p-3 text-sm font-bold text-warning" role="alert">Sua sessão expirou. Entre novamente.</p>
          ) : null}
          {error ? <p className="mt-4 rounded-xl bg-danger-soft p-3 text-sm font-bold text-danger" role="alert">{error}</p> : null}

          <form className="mt-6 grid gap-4" noValidate onSubmit={submit}>
            <Field autoComplete="username" error={userError} id="admin-login" label="Usuário" onChange={(event) => setUserName(event.currentTarget.value)} required value={userName} />
            <Field autoComplete={isSetup ? "new-password" : "current-password"} error={passwordError} id="admin-password" label="Senha" onChange={(event) => setPassword(event.currentTarget.value)} required type="password" value={password} />
            {isSetup ? <Field autoComplete="new-password" error={confirmationError} id="admin-password-confirmation" label="Confirmar senha" onChange={(event) => setConfirmation(event.currentTarget.value)} required type="password" value={confirmation} /> : null}
            <Button disabled={checking || submitting} fullWidth leadingIcon={<LogIn aria-hidden="true" size={17} />} type="submit">{submitting ? "Aguarde…" : isSetup ? "Criar administrador" : "Entrar"}</Button>
          </form>

          <p className="mt-5 text-center text-xs text-muted">
            {isSetup ? <>Já configurou? <Link className="inline-flex min-h-11 items-center font-bold text-brand hover:underline" to="/login">Entrar</Link></> : <>Primeiro acesso? <Link className="inline-flex min-h-11 items-center font-bold text-brand hover:underline" to="/setup">Configurar administrador</Link></>}
          </p>
        </Surface>
      </Page>
    </div>
  );
}
