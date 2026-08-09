# Geração dos pacotes Debian

O projeto gera dois artefatos instaláveis em `dist/`:

- `compasso-client_<versão>_amd64.deb`;
- `compasso-server_<versão>_all.deb`.

## Geração automática

Na raiz do repositório, execute:

```bash
./scripts/build-all-debian-packages.sh
```

O script compila os binários portáteis do cliente e gera os dois pacotes com a
versão declarada em `packaging/debian/control`. Docker é necessário para a
compilação portátil do cliente.

Valide os artefatos com:

```bash
./scripts/test-debian-package.sh
./scripts/test-server-package.sh
```

## Geração manual do cliente

```bash
./scripts/build-portable-client-binaries.sh
./scripts/build-debian-package.sh
./scripts/test-debian-package.sh
```

O primeiro comando compila `tempo-agent`, `tempo-agent-configure` e
`compasso-session-logout` em um contêiner Debian. O segundo monta a árvore
Debian, adiciona os metadados de `packaging/debian/` e chama `dpkg-deb`.

## Geração manual do servidor

```bash
./scripts/build-server-package.sh
./scripts/test-server-package.sh
```

É possível informar outra versão Debian explicitamente:

```bash
./scripts/build-server-package.sh 0.1.0~pilot18
```

O pacote instala os arquivos do Compose em `/opt/compasso-server` e mantém a
configuração editável em `/etc/compasso-server/compasso.env`. A instalação do
`.deb` não inicia contêineres. Depois de revisar a configuração, a implantação
é feita manualmente com:

```bash
sudo /opt/compasso-server/scripts/install-server.sh
```
