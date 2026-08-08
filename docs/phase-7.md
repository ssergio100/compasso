# Fase 7 — servidor, autenticação e painel web

## Implementação

- login administrativo com senha Argon2id;
- sessão expirada, cookie `HttpOnly`/`SameSite=Strict` e `Secure` configurável;
- proteção CSRF em todos os formulários;
- cadastro, renomeação e exclusão de dispositivo;
- cotas independentes para os sete dias;
- criação, edição e exclusão de rotinas, inclusive atravessando meia-noite;
- definição remota do verificador da senha local, sem exibir senha ou hash;
- dashboard “Agora” e histórico administrativo sem dados sensíveis;
- HTML server-side responsivo, sem Node.js ou domínio fixo.

## Testes automatizados

Já executados por `make test`: autenticação correta/incorreta, cookie seguro,
CSRF, expiração, cotas independentes, rotina seg–sex 22:00–08:00, troca de senha
sem vazamento e auditoria.

## Teste visual pelo navegador

Na raiz do repositório:

```bash
cp server/config.example.toml server/config.toml
TEMPO_ADMIN_LOGIN=admin TEMPO_ADMIN_PASSWORD='compasso-teste' \
  go run ./server/cmd/tempo-server -config server/config.toml
```

Abra `http://127.0.0.1:8080` e execute apenas este roteiro:

1. tentar senha errada e confirmar a mensagem genérica;
2. entrar com `admin` / `compasso-teste`;
3. cadastrar “PC do quarto”;
4. definir segunda como `02:00` e terça como `00:45`, salvar e recarregar;
5. criar “Dormir”, seg–sex, `22:00`–`08:00`;
6. definir uma senha local e confirmar que ela não reaparece na tela/histórico;
7. reduzir a janela do navegador e verificar que o painel continua utilizável;
8. sair e confirmar o retorno à tela de login.

Para economizar tokens, responda assim:

```text
login: passou
dispositivo: passou
cotas: passou
rotina: passou
senha local: passou
responsivo: passou
logout: passou
```

Em caso de falha, substitua apenas o item correspondente por `falhou — descrição
curta`. O status online real e os botões de pausa/bloqueio não aparecem nesta
fase; serão adicionados quando funcionarem na fase 8.
