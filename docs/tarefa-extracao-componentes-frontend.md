# Tarefa — extração de componentes do frontend administrativo

**Status:** pendente.

**Interface vigente:** `admin-ui`.

## Contexto

O frontend administrativo ganhou responsabilidades de sessão, comunicação em
tempo real, estado dos computadores, limites, rotinas, identidades visuais,
credenciais e administração. A maior parte dessa composição ainda está
concentrada em `src/App.tsx`.

O comportamento atual está funcional, mas continuar acrescentando recursos no
mesmo arquivo aumentará o risco de regressões, dificultará testes isolados e
criará conflitos frequentes de edição.

## Objetivo

Separar o frontend por funcionalidades, mantendo exatamente a interface e o
contrato atuais. Ao final, `App.tsx` deve cuidar principalmente da composição da
aplicação, autenticação, seleção do computador e navegação entre páginas.

Esta tarefa é uma refatoração estrutural. Não deve redesenhar telas, trocar
imagens, alterar regras de negócio nem modificar a API.

## Estrutura inicial proposta

```text
src/
  features/
    devices/
      DeviceModal.tsx
      DeviceNavigation.tsx
    routines/
      RoutinesPage.tsx
      RoutineModal.tsx
      routineSchedule.ts
    limits/
      LimitsPage.tsx
    administration/
      AdministrationPage.tsx
    now/
      NowPage.tsx
  hooks/
    useDeviceStream.ts
    useNotifications.ts
  visuals.tsx
  api.ts
  types.ts
  App.tsx
```

A estrutura é uma direção, não uma exigência de criar arquivos vazios. Um
arquivo só deve ser extraído quando passar a possuir uma responsabilidade clara.

## Ordem de execução

1. Extrair de `App.tsx` as funções puras de horários, intervalos e conflitos de
   rotina para `features/routines/routineSchedule.ts`, cobrindo-as com testes.
2. Extrair `RoutineModal` e a página/lista de rotinas, preservando criação,
   edição, seleção de ícone, ativação e exclusão.
3. Extrair `DeviceModal` e a navegação de computadores, preservando seleção de
   avatar e estados online, offline, pausado e bloqueado.
4. Extrair as páginas de limites, administração e estado atual sem mover o
   estado compartilhado prematuramente.
5. Extrair a conexão SSE para `useDeviceStream`, mantendo abertura e
   encerramento vinculados ao computador selecionado.
6. Avaliar, somente depois das extrações, se o carregamento e as mutações de
   dispositivos justificam um hook próprio. Não introduzir biblioteca global de
   estado sem necessidade comprovada.

## Regras da refatoração

- Trabalhar somente na interface vigente indicada neste documento.
- Fazer extrações pequenas e compiláveis; evitar uma reescrita integral.
- Preservar mensagens, acessibilidade, comportamento responsivo e modo de
  pré-visualização.
- Não duplicar estado entre `App.tsx` e componentes extraídos.
- Manter chamadas HTTP em `api.ts`; componentes não devem montar URLs.
- Manter tipos compartilhados em `types.ts` enquanto forem realmente comuns.
- Manter o catálogo e os componentes visuais em `visuals.tsx` nesta tarefa.
- Não acrescentar framework de estado, roteador ou biblioteca de formulários
  apenas para realizar a extração.
- Não editar os desenhos nem testar individualmente todos os assets.

## Critérios de aceite

- `App.tsx` deixa de conter a implementação interna dos formulários de
  computador e rotina e das páginas principais.
- Criação e edição de rotina continuam usando o mesmo formulário e persistem o
  `icon_key` escolhido.
- Cadastro e edição de computador continuam persistindo o `avatar_key`.
- Detecção de conflito de horários mantém o comportamento atual, inclusive em
  rotinas que atravessam a meia-noite.
- A troca de computador encerra o stream anterior e abre somente o stream do
  computador selecionado.
- O modo remoto e o modo de pré-visualização continuam funcionando.
- Testes unitários cobrem as regras puras de horários e conflitos.
- Um fluxo representativo em navegador cobre cadastro/edição de identidade de
  computador e criação/edição de rotina; não é necessário repetir o teste para
  cada desenho.
- `npm run build` e a verificação de formatação do repositório passam.

## Fora do escopo

- Redesenhar a interface administrativa.
- Alterar o contrato da API ou o banco de dados.
- Substituir React ou Vite.
- Criar temas visuais novos.
- Eliminar nesta mesma tarefa a repetição das chaves do catálogo entre SQL, Go
  e TypeScript; essa decisão envolve o contrato entre camadas e deve ser tratada
  separadamente.
