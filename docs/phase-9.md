# Fase 9 — segurança e hardening

## Implementação

- revogação explícita do token do dispositivo, com auditoria e efeito imediato;
- HTTPS obrigatório para servidores remotos; HTTP aceito somente em loopback;
- cookies `Secure`, HSTS e limite de cabeçalhos para o servidor de produção;
- payloads limitados, campos inesperados rejeitados e auditoria canonicalizada;
- erros de transporte sanitizados antes de chegar ao log;
- configuração do agente `0600` e diretório/banco `0700`/`0600`, pertencentes a
  `root:root` na instalação;
- unidade systemd com capabilities removidas e isolamento de filesystem,
  dispositivos, kernel, namespaces e diretório pessoal;
- servidor Docker não-root, sem capabilities, filesystem somente leitura e
  porta publicada apenas no loopback do host;
- como implementação intermediária, a senha inicial do administrador esteve
  disponível por Docker secret; o pacote atual cria o primeiro acesso somente
  depois da instalação, pela interface;
- exemplos separados de servidor Docker e Cloudflare Tunnel para
  `apicompasso.smresume.com` e `admcompasso.smresume.com`.

A configuração de ingress segue o formato atual documentado pela
[Cloudflare](https://developers.cloudflare.com/tunnel/advanced/local-management/configuration-file/),
incluindo uma regra final de fallback. O código da aplicação não contém nomes
de domínio fixos.

## Validações automatizadas

`make test` cobre:

- token revogado rejeitado pela autenticação;
- token, senha e verificador ausentes de erros e auditoria;
- rejeição de HTTP remoto e aceitação de HTTPS/loopback;
- limites e campos inesperados no heartbeat;
- corpus inicial dos fuzz tests do servidor e agente;
- diretivas obrigatórias do hardening systemd;
- sintaxe do instalador e do Compose;
- execução não-root e exclusão de configurações/secrets do contexto Docker.

O `systemd-analyze security --offline=yes` classificou a unidade com exposição
`3.2 OK` no ambiente de desenvolvimento.

## Construção segura da imagem

- [x] Imagem `compasso-tempo-server:latest` construída com sucesso no Zorin.

Este comando somente baixa as camadas necessárias e constrói a imagem; ele não
inicia o servidor e não conflita com a porta 8080 usada pelo qBittorrent:

```bash
cd /home/sergio/projetos/compasso
docker compose build compasso-api
```

## Instalação segura do agente no Zorin

- [x] Agente instalado como serviço root com configuração `0600`, estado `0700` e sem alteração do PAM.
- [x] Usuário comum sem autenticação administrativa não lê/altera o estado e não consegue parar o serviço.
- [x] Serviço permaneceu ativo e o dispositivo retornou a `ONLINE` no painel.

Não instale o gate PAM durante este teste. Antes de iniciar, mantenha a política
remota em **Pausar vigilância** para impedir logout acidental.

1. Encerre o agente iniciado manualmente na fase 8 com `Ctrl+C`.
2. Reinicie o servidor com o código atual e mantenha a vigilância pausada.
3. No painel, revogue o token antigo e gere uma credencial nova. Não copie esse
   novo token para `agent/config.toml`.
4. Gere o pacote como usuário comum:

   ```bash
   cd /home/sergio/projetos/compasso
   make package-client
   ```

5. Instale o pacote e abra **Compasso** para informar usuário, URL, `device_id`
   e o novo token pela engrenagem:

   ```bash
   sudo apt install ./dist/compasso-client_0.1.0~pilot19_amd64.deb
   ```

   O assistente valida a configuração e só confirma o pareamento depois do
   primeiro heartbeat aceito.

6. Como usuário comum, valide as permissões:

   ```bash
   if test -r /etc/tempo-agent/config.toml || test -w /etc/tempo-agent/config.toml; then
     echo "permissões: falhou"
   elif test -r /var/lib/tempo-agent/tempo-agent.db || test -w /var/lib/tempo-agent/tempo-agent.db; then
     echo "permissões: falhou"
   else
     echo "permissões: passou"
   fi
   ```

7. Tente parar o serviço sem autenticação administrativa e confirme que ele
   permanece ativo:

   ```bash
   if systemctl --no-ask-password stop tempo-agent.service; then
     echo "bloqueio do serviço: falhou"
     sudo systemctl start tempo-agent.service
   elif systemctl is-active --quiet tempo-agent.service; then
     echo "bloqueio do serviço: passou"
   else
     echo "bloqueio do serviço: falhou — serviço inativo"
   fi
   ```

8. Atualize o painel após até 10 segundos e confirme que o dispositivo está
   `ONLINE`. Mantenha a vigilância pausada até terminar os testes.

Responda somente:

```text
imagem Docker: passou
instalação: passou
permissões: passou
bloqueio do serviço: passou
online: passou
```
