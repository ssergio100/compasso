# Atualização manual do servidor

> Atualize primeiro os agentes. O agente novo aceita o servidor antigo, mas o
> servidor novo recusa com HTTP 426 qualquer bônus pendente para agente que não
> anuncie `X-Compasso-Protocol-Version: 2`. Isso evita saldo exibido sem ter
> sido aplicado.

Substitua `VERSAO`, `USUARIO`, `SERVIDOR` e `DOMINIO` pelos valores da
implantação.

## 1. Gerar o pacote

Na raiz do repositório:

```bash
./scripts/build-server-package.sh VERSAO
./scripts/test-server-package.sh
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

#ou


