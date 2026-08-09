# Etapa 3 — pacote completo do servidor com Docker Compose

## Objetivo

Instalar todo o servidor Compasso em um Debian 13 sem interface gráfica usando
um único pacote e um único `compose.yaml`. A instalação deve respeitar a
organização já adotada no servidor Dell:

```text
/srv/docker/compose/compasso     projeto e compose.yaml
/srv/docker/volumes/compasso     dados persistentes
/srv/docker/backups/compasso     backups
```

O host precisa apenas do Docker Engine e do plugin Docker Compose; o instalador
deve detectar sua ausência, pedir autorização e instalá-los pelo gerenciador de
pacotes.

Inventário confirmado no Dell em 08/08/2026, útil para a implantação doméstica
mas sem criar dependência no pacote:

- [x] Docker Compose v5.3.1 disponível.
- [x] Rede bridge externa `shared-services` existente.
- [x] Nenhum contêiner `cloudflared` em execução.
- [x] Configuração de túnel local encontrada em `/srv/cloudflare/config.yml`,
  acompanhada do arquivo de credencial JSON.
- [x] `cloudflared` confirmado como serviço systemd ativo e habilitado.
- [x] Portas `8181` para API e `8182` para frontend confirmadas como livres.

O pacote reúne os componentes, mas mantém frontend e backend desacoplados em
contêineres diferentes. Assim uma atualização visual não recompila nem reinicia
a API.

## Topologia

```text
Navegador -----------------> compasso-admin-ui :8182
                                  |
                                  | JSON
                                  v
Agentes Linux ------------> compasso-api :8181
                                  |
                                  v
                         bind persistente / SQLite
```

Serviços obrigatórios do Compose do Compasso:

- `compasso-api`: backend Go, sem templates ou arquivos do frontend;
- `compasso-admin-ui`: servidor estático da interface administrativa;
- armazenamento bind em `/srv/docker/volumes/compasso` para o banco SQLite.

O Compose publica API e frontend diretamente nas interfaces e portas definidas
no `.env`. O padrão `0.0.0.0` permite uso imediato pela rede local. Quem desejar
um proxy, túnel ou acesso somente local pode trocar o bind para `127.0.0.1` ou
outra interface sem alterar imagens ou código da aplicação.

## Segurança e configuração

- instalação sem credenciais; primeiro administrador criado posteriormente na
  interface e armazenado somente como hash Argon2id;
- banco persistente fora da camada gravável do contêiner;
- API executada como usuário sem privilégios;
- frontend executado como usuário sem privilégios;
- cookies `HttpOnly`, proteção CSRF e CORS restrito ao mesmo hostname por
  padrão; quando houver HTTPS, `Secure` e uma origem administrativa explícita
  são habilitados por `.env`;
- imagens com versões fixadas e healthchecks;
- nenhum segredo, token de túnel ou domínio fixado nas imagens.

## Entregas

### 3.1 — API independente

- [x] Completar os endpoints JSON administrativos.
- [x] Implementar login, sessão, logout e CSRF em JSON.
- [x] Restringir CORS à origem administrativa configurada.
- [x] Remover templates, arquivos estáticos e `assets_directory` do backend.
- [x] Confirmar que o backend compila e testa sem o diretório do frontend.

### 3.2 — frontend independente

- [x] Criar `admin-ui/` com HTML, CSS e JavaScript próprios.
- [x] Preservar inicialmente o visual e as funções já aprovadas.
- [x] Consumir somente a API configurada em runtime.
- [x] Criar testes com API simulada.
- [x] Criar imagem própria que não contém o backend.

### 3.3 — Compose e instalação

- [x] Manter túnel, proxy, VPN, DNS e TLS fora do instalador da aplicação.
- [x] Criar `compose.yaml` com API, frontend e armazenamento bind, sem
  dependência de serviços externos de exposição.
- [x] Criar `.env.example` sem segredos.
- [x] Criar instalador que não mistura implantação com configuração do primeiro
  administrador.
- [x] Criar configuração inicial pela interface, encerrada após o primeiro
  administrador.
- [x] Detectar e, após autorização, instalar Docker e Compose ausentes.
- [x] Criar comandos simples para instalar, atualizar, verificar e restaurar.
- [x] Criar backup consistente do SQLite antes de atualizações.

### 3.4 — validação

- [x] `docker compose config` não revela senha ou token.
- [x] Pacote inicia API e painel diretamente por IP e portas configuráveis, sem
  túnel, proxy ou DNS (`0.0.0.0:8281/8282` no smoke test local).
- [x] Modo padrão aceita o frontend no mesmo hostname e bloqueia origem com
  hostname diferente.
- [x] Instalação vazia inicia sem credenciais, oferece configuração posterior,
  autentica o primeiro administrador e rejeita uma segunda configuração
  (`201`, sessão autenticada e depois `409` no smoke test do `pilot2`).
- [x] No Dell Debian 13, o `pilot2` foi instalado sem credenciais, abriu pela
  LAN em `http://192.168.18.10:8182`, concluiu o primeiro acesso pela interface
  e passou pelo login posterior (teste real informado pelo responsável).
- [ ] Todos os contêineres ficam saudáveis após reboot.
- [x] API não entrega HTML; somente a imagem administrativa serve o frontend.
- [x] Frontend não contém banco, credenciais ou binário do servidor.
- [x] Alterar e reconstruir somente o frontend não reinicia a API.
- [ ] Atualização preserva banco, configuração e segredos.
- [ ] Restauração recupera o banco anterior.

## Critério de pronto

- [x] No Dell Debian 13, o responsável extrai um pacote em
  `/srv/docker/compose/compasso`, executa um único
  instalador sem fornecer credenciais ou decisões de infraestrutura.
- [ ] O instalador apresenta e instala dependências ausentes após autorização.
- [ ] Após reiniciar o servidor, painel e API voltam automaticamente.
- [x] Frontend e backend podem receber versões independentes dentro do mesmo
  pacote Compose.

## Decisão de implantação

“Juntos” significa distribuídos e iniciados pelo mesmo pacote Compose. Não
significa executar frontend e backend no mesmo contêiner. Essa fronteira é
necessária para que a interface permaneça realmente substituível e atualizável
de forma independente. A forma de acesso ao host não faz parte desta unidade:
LAN, Cloudflare Tunnel, proxy reverso, VPN ou nenhuma exposição são igualmente
compatíveis e escolhidos posteriormente pelo responsável pela infraestrutura.
