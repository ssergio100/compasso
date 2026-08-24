# Publicação do servidor

> Atualize primeiro os agentes. O agente novo aceita o servidor antigo, mas o
> servidor novo recusa com HTTP 426 qualquer bônus pendente para agente que não
> anuncie `X-Compasso-Protocol-Version: 2`. Isso evita saldo exibido sem ter
> sido aplicado.

## Fluxo recomendado

Na raiz do repositório, execute:

```bash
./scripts/publish-server.sh
```

O publicador mostra o último piloto gerado e pergunta a nova versão, o endereço
do servidor, usuário e porta SSH, chave opcional e URL do healthcheck. Antes de
alterar o servidor ele:

- avisa sobre alterações locais ainda não commitadas;
- executa os testes, gera o `.deb` e valida seu conteúdo;
- verifica acesso SSH, ferramentas remotas, espaço disponível e autorização
  `sudo`;
- compara a versão nova com a instalada e exige confirmação para reinstalação
  ou downgrade;
- confere o SHA-256 depois do envio.

A atualização remota sempre cria um backup dos dados antes de reconstruir e
reiniciar a API. O script usa um terminal SSH para que a senha do `sudo`, quando
necessária, seja digitada diretamente pelo operador.

Para apenas gerar e validar um pacote:

```bash
./scripts/publish-server.sh --build-only
```

As opções disponíveis podem ser consultadas com
`./scripts/publish-server.sh --help`. Nenhuma senha é salva pelo script.

## Fluxo manual de contingência

Substitua `VERSAO`, `USUARIO`, `SERVIDOR` e `DOMINIO` pelos valores da
implantação.

## 1. Gerar o pacote

Na raiz do repositório:

```bash
./scripts/build-server-package.sh VERSAO
./scripts/test-server-package.sh dist/compasso-server_VERSAO_all.deb
sha256sum dist/compasso-server_VERSAO_all.deb
```

Anote o hash apresentado.

## 2. Transferir

```bash
scp dist/compasso-server_VERSAO_all.deb USUARIO@SERVIDOR:/tmp/
ssh USUARIO@SERVIDOR
sha256sum /tmp/compasso-server_VERSAO_all.deb
```

Confirme que os hashes são iguais.

## 3. Instalar

```bash
sudo apt install /tmp/compasso-server_VERSAO_all.deb
sudo nano /etc/compasso-server/compasso.env
```

Se solicitado, preserve o `compasso.env` existente. Altere somente:

```text
COMPASSO_VERSION=VERSAO
```

Atualize os contêineres:

```bash
sudo /opt/compasso-server/scripts/update-server.sh
```

O script faz backup do banco, constrói a imagem, reinicia a API e executa o
healthcheck.

## 4. Conferir

```bash
sudo /opt/compasso-server/scripts/status-server.sh
curl -fsS https://DOMINIO/healthz
```

O endpoint deve responder `{"status":"ok"}`.

## Observação

O pacote atualiza a API. A interface administrativa é publicada separadamente.

