# Controle de Tempo Linux

Monorepo do Controle de Tempo Linux, um sistema de cotas e rotinas para sessões
Linux que continua aplicando a última política válida sem conexão com o servidor.

## Componentes

- `agent`: daemon privilegiado, motor de política e persistência local.
- `server`: painel administrativo, API de sincronização e persistência central SQLite.
- `protocol`: contrato versionado de heartbeat compartilhado pelo servidor e agente.
- `local-ui`: interface GTK para intervenções locais.
- `docs`: documentação de desenvolvimento e decisões técnicas.
- `packaging`: arquivos de instalação e empacotamento Linux.

## Desenvolvimento

Pré-requisitos: Go, `make` e o executável `sqlite3`.

```bash
make test
```

Consulte [docs/development.md](docs/development.md) para os comandos e as
convenções do projeto. A especificação funcional de referência está em
`Controle_Tempo_Linux_Especificacao_v0.1.md`.
