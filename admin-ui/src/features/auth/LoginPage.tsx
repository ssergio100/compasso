import { useState } from "react";
import { Brand } from "../../components";

export function LoginPage({ onLogin }: { onLogin: (login: string, password: string) => Promise<boolean> }) {
  const [login, setLogin] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  return <main className="login"><section><Brand /><h1>Acesso administrativo</h1><p>Administre limites e rotinas.</p><form onSubmit={async (event) => { event.preventDefault(); setBusy(true); try { if (!await onLogin(login, password)) setError("Usuário ou senha inválidos."); } catch { setError("Não foi possível entrar."); } finally { setBusy(false); } }}><label>Usuário<input autoComplete="username" value={login} onChange={(event) => setLogin(event.target.value)} /></label><label>Senha<input autoComplete="current-password" type="password" value={password} onChange={(event) => setPassword(event.target.value)} /></label>{error && <p className="error">{error}</p>}<button className="primary-button" disabled={busy || !login || password.length < 6}>{busy ? "Entrando…" : "Entrar"}</button></form></section></main>;
}
