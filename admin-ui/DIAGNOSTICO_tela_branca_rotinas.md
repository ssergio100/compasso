# Diagnóstico — tela branca ao criar rotina

**Interface:** `admin-ui`
**Fluxo:** Rotinas → Nova rotina → **Criar rotina**
**Sintoma:** a tela fica totalmente branca após o cadastro, embora a rotina seja persistida no servidor.

## Mecânica do sintoma

Tela branca em React = exceção não tratada durante o render, que desmonta a árvore
inteira. O app é montado sem ErrorBoundary (`src/main.tsx`), portanto qualquer erro
de render vira página em branco, sem mensagem e sem recuperação.

A rotina continua salva porque o erro não está no POST:

1. `POST /api/v1/admin/devices/:id/routines` retorna sucesso (rotina gravada);
2. update otimista + fechamento do modal + toast (`src/App.tsx:90`);
3. `load()` busca novamente todos os dispositivos e detalhes;
4. **é no re-render com esses dados que a exceção estoura** — depois que o servidor
   já confirmou a gravação.

## Cenários verificados (Chrome headless via CDP)

| Cenário | Resultado |
| --- | --- |
| Dev + servidor local (árvore de trabalho atual) | OK, sem exceções |
| Build de produção (`npm run build`) + API real | OK |
| Dev + servidor na versão HEAD (sem endpoint SSE) | OK |
| Múltiplos computadores; rotina virada da noite; dia inteiro; conflito de horário; toggle e exclusão; troca de dispositivo | OK |

Conclusão: com dados limpos o fluxo não falha localmente. O gatilho é específico do
ambiente de teste (dado devolvido pelo servidor ou timing), mas o código tem pontos
que transformam qualquer payload inesperado em tela branca.

## Causas prováveis (em ordem)

1. **Payload inesperado no `load()` posterior à gravação**, consumido sem blindagem:
   - `routine.days.map(...)` e `device.routines.length` em `src/App.tsx:148`;
   - desconstrução de `detail.policy.*` em `src/api.ts:41-51`.
   Se o servidor real devolver `routines: null`, campo ausente ou policy parcial,
   o crash ocorre exatamente nesse momento.
2. **Evento do stream SSE chegando durante a transição pós-save**
   (`src/App.tsx:40-48`) — código novo, ainda não commitado.
3. **`crypto.randomUUID()` indisponível** fora de contexto seguro (acesso por IP em
   HTTP puro). Nesse caso, porém, o modal exibiria o erro em vez de telar em branco.

Observação: o formato de `days` no servidor é estável (`[7]bool` desde `ec19d86`),
o que reduz — mas não elimina — a hipótese 1 para versões empacotadas recentes.

## Correção sugerida (defesa em profundidade)

### 1. ErrorBoundary envolvendo o conteúdo principal

Em vez de tela branca, um estado recuperável com ação "Recarregar", além de logar o
erro real no console — o que também revela a causa exata no ambiente afetado.

```tsx
class ErrorBoundary extends Component<{ children: ReactNode }, { error: Error | null }> {
  state = { error: null as Error | null };
  static getDerivedStateFromError(error: Error) { return { error }; }
  componentDidCatch(error: Error) { console.error("Falha na interface:", error); }
  render() {
    if (this.state.error) {
      return <div className="center-state" role="alert">
        <Brand /><h1>Algo saiu do trilho</h1>
        <p>Não foi possível exibir esta tela.</p>
        <button className="primary-button" onClick={() => location.reload()}>
          <RefreshCw size={18} />Recarregar
        </button>
      </div>;
    }
    return this.props.children;
  }
}
```

Uso em `src/main.tsx`: `<StrictMode><ErrorBoundary><App /></ErrorBoundary></StrictMode>`.

### 2. Normalizar o payload na camada de API (`src/api.ts`)

Garantir arrays antes dos dados chegarem ao render:

- `routines: detail.policy.routines ?? []`;
- cada rotina com `days` normalizado para array de 7 booleanos;
- `weekly_quota_seconds ?? []`.

Assim, payload nulo/parcial do servidor degrada graciosamente (lista vazia) em vez
de derrubar a aplicação.

### 3. Fallback para geração de id

Substituir o uso direto de `crypto.randomUUID()` por utilitário com fallback
(ex.: `Date.now()` + aleatório) para ambientes sem contexto seguro.

## Impacto

- Nenhuma mudança visual ou de interação — padrões atuais preservados.
- Falhas futuras deixam de ser "tela morta": passam a mostrar mensagem coerente,
  manter o resto do app navegável quando possível e registrar a causa real.

## Próximo passo

Implementar os três itens neste protótipo após confirmação. Com o ErrorBoundary em
plano, se o problema voltar, o console exibirá a exceção original e permite fechar
o diagnóstico definitivo contra o servidor em `192.168.18.10:8181`.
