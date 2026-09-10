# Compasso

Controle familiar de tempo para computadores Linux, com cotas diárias,
rotinas, bônus e intervenções administrativas. O agente aplica localmente a
última autorização válida, inclusive durante falhas temporárias de rede.

> **Estado do projeto:** versão `0.1.0-dev`, adequada para desenvolvimento e
> pilotos controlados. O cliente está direcionado inicialmente a sistemas
> Debian/Ubuntu derivados, `amd64`, com systemd/logind. Ainda não é uma solução
> pronta para ambientes críticos ou para múltiplas famílias em um serviço
> compartilhado.

## O que já funciona

- cotas diferentes para cada dia e rotinas recorrentes, inclusive atravessando
  a meia-noite;
- bloqueio da sessão gráfica sem encerrar aplicativos;
- operação local com SQLite quando o servidor está indisponível;
- bônus local protegido por senha e bônus remoto com confirmação do agente;
- pausa, retomada, bloqueio e desbloqueio pelo painel;
- pareamento individual de dispositivos e sincronização idempotente;
- painel administrativo responsivo, histórico de atividades e atualizações por
  Server-Sent Events (SSE);
- pacotes Debian separados para cliente e servidor.

## Visão geral

| Componente | Responsabilidade | Tecnologia |
| --- | --- | --- |
| `agent/` | aplicar regras, observar a sessão e persistir o estado local | Go, systemd/logind, SQLite |
| `server/` | autenticação, API, sincronização e estado central | Go, HTTP, SQLite |
| `admin-ui/` | única interface administrativa vigente | React, TypeScript, Vite |
| `local-ui/` | configuração do agente e concessão local de tempo | Python, GTK 4 |
| `protocol/` | contrato compartilhado do heartbeat | Go |

O [mapa do repositório](docs/repository-map.md) explica os diretórios e pontos
de entrada em mais detalhes.

## Visualização rápida do painel

O modo demonstrativo não precisa do servidor e é o caminho mais curto para
conhecer a interface:

```bash
cd admin-ui
npm ci
npm run dev
```

Abra `http://127.0.0.1:4175/?preview=visuals`. Os dados desse modo são fictícios
e nenhuma alteração é persistida. Consulte o [guia de demonstração](docs/demo.md)
para preparar capturas de tela.

## Desenvolvimento

Pré-requisitos para a verificação completa: Linux, Go, compilador C, GNU Make,
SQLite CLI, Python 3, Node.js e npm. PyGObject/GTK 4 é necessário para abrir as
janelas locais, mas os testes de lógica não iniciam a interface gráfica.

```bash
git clone https://github.com/ssergio100/compasso.git
cd compasso
npm ci --prefix admin-ui
make test
```

O comando valida formatação e código Go, testes Go e Python, frontend,
migrações, empacotamento e builds. Veja [Desenvolvimento](docs/development.md)
e [Como contribuir](CONTRIBUTING.md).

## Instalação e distribuição

- [instalação do cliente Linux](docs/client-installation.md);
- [instalação da API do servidor](docs/server-installation.md);
- [geração dos pacotes Debian](docs/debian-packaging.md);
- [implantação da interface administrativa](deploy/README.md).

Artefatos gerados ficam em `dist/` e não são versionados. Configurações locais,
bancos, chaves e notas privadas também ficam fora do Git; nunca inclua
credenciais em exemplos ou relatórios.

## Documentação

O [índice de documentação](docs/README.md) reúne guias vigentes e referências
de arquitetura; o [histórico de mudanças](CHANGELOG.md) resume a evolução
pública. Para reportar uma vulnerabilidade, siga [SECURITY.md](SECURITY.md).

## Aviso de segurança

O agente roda com privilégios de sistema e pode bloquear uma sessão gráfica.
Teste alterações de política e empacotamento primeiro em uma máquina virtual ou
em um equipamento de laboratório com acesso de recuperação disponível.

## Licença

O código, a documentação e os recursos produzidos para o Compasso são
distribuídos sob a [GNU Affero General Public License v3.0 ou posterior](LICENSE)
(`AGPL-3.0-or-later`). Ao executar uma versão modificada para usuários por uma
rede, disponibilize a eles o código-fonte correspondente, conforme os termos da
licença.

As atribuições e licenças dos componentes de terceiros estão em
[THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
