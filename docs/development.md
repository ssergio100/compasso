# Desenvolvimento

## Requisitos

- Linux amd64;
- Go 1.26.x como versão-alvo do projeto;
- GNU Make;
- SQLite 3 com o executável `sqlite3` no `PATH`.
- compilador C compatível com CGO para o driver SQLite do agente.

O código do motor também permanece compatível com Go 1.18 para permitir os
testes no ambiente de desenvolvimento atual.

## Comandos

```bash
make fmt             # formata fontes Go
make lint            # valida gofmt e executa go vet
make build-agent     # gera bin/tempo-agent
make test-go         # executa testes Go
make test-migrations # cria bancos temporários a partir do zero
make test            # executa todas as validações e compila
```

Os bancos e arquivos criados durante o desenvolvimento devem ficar em `var/`,
que não é versionado. Copie os arquivos `config.example.toml` de cada componente
para uma configuração local ignorada pelo Git quando esses serviços forem
implementados.

A integração de sessões da fase 3 usa `loginctl`, disponível nas distribuições
com systemd. Os testes do pacote `agent/daemon` usam uma implementação simulada:
eles nunca encerram a sessão da máquina de desenvolvimento.

## Migrações

Cada componente mantém migrações próprias em `storage/migrations`. Os arquivos
usam o formato `NNNN_descricao.sql`, são imutáveis depois de publicados e devem:

1. ser aplicáveis em ordem lexical a um banco vazio;
2. registrar sua versão em `schema_migrations`;
3. atualizar `PRAGMA user_version` para a mesma versão;
4. preservar a integridade referencial.

Uma mudança de esquema exige um arquivo novo; migrações já distribuídas não são
editadas.
