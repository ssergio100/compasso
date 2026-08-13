# Administração mobile-first

Base componentizada da nova interface administrativa. A aplicação usa React,
TypeScript, Tailwind CSS e React Router. Os dados ainda são fictícios e a API de
produção em `admin-ui/` não foi substituída.

Para instalar as dependências e abrir localmente a partir da pasta do
protótipo:

```bash
cd docs/prototypes/admin-ui-mobile
npm install
npm run dev
```

Depois acesse `http://127.0.0.1:4173`.

O Vite oferece atualização automática, e os ícones são componentes do
`lucide-react`. A organização principal é:

```text
src/
├── app/          # rotas e estado compartilhado
├── components/   # componentes reutilizáveis
├── data/         # dados simulados substituíveis por API
├── domain/       # contratos de clientes, limites e rotinas
├── features/     # páginas e regras por funcionalidade
├── layouts/      # cabeçalho, página e navegação do cliente
└── styles/       # tokens e estilos globais
```

Comandos de validação:

```bash
npm test
npm run typecheck
npm run build
```

Fluxos representados:

- seleção inicial de clientes;
- primeiro acesso, login, sessão expirada e logout;
- cadastro de cliente com validação e criação no estado compartilhado;
- cartão principal do cliente com estado, saldo e ações rápidas;
- navegação inferior para Agora, Limites, Rotinas e Administração;
- configuração da cota por dia com seletor único de tempo e atalhos para
  liberar ou bloquear o dia, usando rascunho antes de salvar;
- criação e edição de rotina em duas etapas: nome/ícone e horário/dias;
- ativação, desativação e exclusão de rotinas, com detecção de conflitos
  semanais e estado vazio orientativo;
- edição do nome do cliente, pareamento com `device_id` e `device_token`, senha
  local e remoção segura;
- estado detalhado do agente, sessão gráfica, monitoramento e contagem;
- retorno de comandos pendentes, aplicados ou com possibilidade de nova tentativa;
- antecedência do aviso de tempo e histórico de eventos administrativos;
- componentes compartilhados para loading, erro, tentar novamente e estado vazio;
- adaptação de uma para duas colunas em telas maiores.

Rotas disponíveis:

- `/` — clientes;
- `/login` — acesso administrativo;
- `/setup` — configuração do primeiro administrador;
- `/clients/new` — novo cliente;
- `/clients/:clientId/now` — uso atual;
- `/clients/:clientId/limits` — limites;
- `/clients/:clientId/routines` — rotinas;
- `/clients/:clientId/routines/new` — nova rotina;
- `/clients/:clientId/routines/:routineId/edit` — editar rotina;
- `/clients/:clientId/administration` — administração técnica;
- `/clients/:clientId/administration/name` — editar nome;
- `/clients/:clientId/administration/credential` — gerenciar o pareamento do agente;
- `/clients/:clientId/administration/password` — configurar senha local;
- `/clients/:clientId/administration/warning` — configurar o aviso de tempo;
- `/clients/:clientId/administration/history` — consultar o histórico.

## Estados assíncronos

As ações direcionadas ao computador apresentam três estados: aguardando
sincronização, aplicado e erro recuperável. Quando o agente está offline, a
alteração permanece pendente no Compasso até a próxima conexão.

## Semântica das rotinas

- O dia selecionado é sempre o dia em que a rotina começa.
- Se o horário final for anterior ao inicial, a rotina termina no dia seguinte.
  Por exemplo, segunda-feira das 22:00 às 07:00 termina na terça-feira às
  07:00.
- O início pertence à rotina e o instante final não pertence. Assim, uma rotina
  pode terminar às 07:00 e outra começar às 07:00 sem conflito.
- Rotinas não podem se sobrepor, inclusive na meia-noite e na passagem de
  domingo para segunda-feira.
