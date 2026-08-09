# Plano de adequação — frontend administrativo independente

## Objetivo

Separar completamente a interface administrativa do servidor. O backend deve
oferecer somente regras de negócio, persistência, autenticação e uma API JSON.
O frontend deve ser uma aplicação independente, com código, testes, build e
implantação próprios.

Essa separação permite melhorar ou substituir a interface por React, Vue ou
outra tecnologia sem alterar, recompilar ou reiniciar o servidor Go.

## Fronteiras obrigatórias

### Backend

- expõe a API versionada em `/api/v1`;
- autentica administradores e dispositivos;
- valida autorização, CSRF e todos os dados recebidos;
- executa regras de negócio e acessa o banco;
- responde com JSON e códigos HTTP consistentes;
- não lê templates, não renderiza HTML e não serve arquivos do frontend;
- não depende da existência do diretório ou do build do frontend.

### Frontend administrativo

- vive em `admin-ui/`, separado do código Go;
- acessa o backend exclusivamente pela API JSON;
- não acessa banco nem importa código interno do servidor;
- possui configuração própria para a URL da API;
- pode ser compilado, testado e implantado sem recompilar o backend;
- começa com JavaScript simples para reaproveitar a interface atual e reduzir
  custo; a adoção futura de React será uma decisão exclusiva do frontend.

## Topologia lógica

```text
Navegador
    |
    v
endereço configurado          frontend estático independente
    |
    | HTTPS + JSON
    v
endereço configurado          backend Go / API v1
    |
    v
SQLite do servidor
```

Frontend e API podem usar IP, hostname, portas ou domínios diferentes. Túnel,
proxy, VPN, DNS e TLS são decisões de infraestrutura posteriores. Nenhuma delas
poderá criar dependência entre o código do frontend e o backend.

## Migração rápida e de baixo custo

### Etapa 1 — fechar o contrato JSON

- [x] Inventariar todas as operações usadas pelo painel atual.
- [x] Criar os endpoints JSON ainda ausentes para sessão, dispositivos,
  política, rotinas, senha local, bônus, pausa, retomada, bloqueio e histórico.
- [x] Definir formatos consistentes para sucesso, validação e erro.
- [x] Manter o contrato versionado em `/api/v1`.
- [x] Criar testes HTTP do contrato sem carregar qualquer HTML.

### Etapa 2 — autenticação entre origens configuráveis

- [x] Usar cookie de sessão `Secure`, `HttpOnly` e restrito ao host da API.
- [x] Permitir CORS com credenciais somente para a origem administrativa
  configurada, nunca com origem curinga.
- [x] Exigir token CSRF nas operações que alteram estado.
- [x] Validar `Origin` nas requisições administrativas mutáveis.
- [ ] Confirmar login, expiração e logout usando frontend e API em origens
  diferentes durante o desenvolvimento.

### Etapa 3 — extrair a interface atual

- [x] Criar `admin-ui/` sem dependência de pacotes do servidor Go.
- [x] Transferir HTML, CSS e JavaScript atuais sem redesenhar as telas.
- [x] Substituir templates e formulários server-side por chamadas à API JSON.
- [x] Centralizar chamadas HTTP e tratamento de erros em um módulo cliente da
  API com nomes claros.
- [x] Manter atualização automática do estado e dos contadores.
- [x] Permitir configurar a URL da API sem editar o código-fonte.

### Etapa 4 — execução e distribuição independentes

- [x] Criar um comando de desenvolvimento somente para o frontend.
- [x] Criar uma imagem de contêiner somente para o frontend estático.
- [x] Remover templates e arquivos estáticos da imagem do backend.
- [x] Documentar endereços independentes e configuráveis para frontend e API.

### Etapa 5 — retirar a implementação acoplada

- [ ] Validar paridade funcional da nova interface pelo navegador.
- [x] Remover renderização de templates e rotas de formulário do servidor Go.
- [x] Remover `assets_directory` e funções auxiliares de HTML do backend.
- [x] Confirmar que os testes do servidor passam sem o diretório `admin-ui/`.

## Critérios de pronto

- [x] O backend compila, inicia e executa todos os testes sem arquivos do
  frontend presentes.
- [x] O frontend executa seus testes com uma API simulada, sem iniciar o
  servidor real.
- [x] Uma alteração visual é publicada sem recompilar ou reiniciar o backend.
- [x] Backend e frontend geram imagens e versões independentes.
- [x] O navegador não recebe HTML renderizado pelo servidor Go.
- [x] Todas as operações administrativas passam somente pela API JSON.

## Fora do escopo desta adequação

- redesenhar a identidade visual;
- escolher React ou outro framework agora;
- decidir o provedor definitivo de hospedagem do frontend;
- adicionar WebSocket ou MQTT.

Essas escolhas poderão ser feitas depois sem alterar o contrato entre frontend
e backend.
