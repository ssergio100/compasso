# Fase 10 — teste ponta a ponta e piloto

## Implementação

- alertas de cota e rotina são enviados à sessão gráfica do usuário controlado;
- o envio de um alerta possui timeout e nunca interrompe a aplicação da política;
- a semana acelerada cobre as sete cotas diárias, rotina atravessando meia-noite
  e vigilância pausada;
- uma senha nova só substitui a antiga depois de sincronização bem-sucedida;
- o pacote piloto inclui instalação, desinstalação e recuperação temporizada;
- a instalação não ativa o gate PAM automaticamente;
- HTML e CSS do painel são arquivos externos configurados por
  `assets_directory`; em desenvolvimento, mudanças aparecem ao atualizar a
  página, sem recompilar ou reiniciar o servidor. Essa separação permite trocar
  a interface por React ou outra tecnologia sem embuti-la no binário Go.
- [x] Recarregamento de template externo sem reinício validado em teste automatizado.

## Ordem segura dos ensaios no Zorin

Os testes são divididos em etapas. Não avance para PAM/logout antes de validar
interface, alerta e recuperação. Mantenha **Pausar vigilância** ativo enquanto
instala ou atualiza componentes.

### 1. Instalar os componentes sem PAM

- [x] Componentes piloto instalados no Zorin e `tempo-agent.service` confirmado como ativo.

```bash
cd /home/sergio/projetos/compasso
make build-agent
sudo ./scripts/install-pilot-components.sh
```

Esse passo atualiza o serviço já instalado, instala o diálogo GTK e os comandos
de recuperação. Ele não modifica `/etc/pam.d/gdm-password`.

### 2. Validar interface e alerta sem risco de bloqueio de login

- [x] Diálogo GTK validado no Zorin com D-Bus disponível, senha incorreta, rate limit e bônus correto de 15 minutos.
- [x] Painel mantém a cota diária fixa e adiciona tempo extra somente ao tempo restante, sem campo separado de bônus (teste automatizado com cota 08:00 e restante 08:10 após adicionar 10 minutos).
- [x] Contadores visuais mostram segundos: tempo restante decresce e tempo usado aumenta quando o cliente está online e a vigilância está contabilizando.
- [x] Endpoint JSON administrativo fornece status vivo para a interface atual e para uma futura migração para React.
- [x] Contadores consultam o servidor em segundo plano a cada dois segundos e incorporam tempo extra local sem recarregar a página.
- [x] Tempo extra concedido no agente apareceu automaticamente no painel do Zorin, sem atualização manual da página.
- [x] Mensagens administrativas de sucesso desaparecem automaticamente e não reaparecem ao atualizar a página.
- [x] Alerta visual validado na sessão gráfica do Zorin antes do fim da cota.

Abra **Adicionar tempo — Compasso** pelo menu do Zorin. O ensaio de alerta será
feito com uma cota curta e vigilância retomada, mas o PAM continuará ausente.
Depois que o aviso aparecer, pause novamente a vigilância no painel.

### 3. Ativar o gate somente com recuperação agendada

Antes de instalar o gate, agende uma recuperação que restaura o PAM, para o
agente e permite novo login mesmo se o servidor ficar indisponível:

```bash
sudo /usr/sbin/tempo-schedule-recovery 5min
sudo /usr/sbin/tempo-pam-setup -action install
```

Nunca cancele o temporizador antes de confirmar que o login gráfico voltou a
funcionar. O estado pode ser visto com:

```bash
systemctl status compasso-pilot-recovery.timer --no-pager
```

### 4. Desinstalação e rollback

```bash
sudo /usr/sbin/tempo-uninstall
```

O desinstalador restaura o arquivo PAM original antes de remover o helper. A
configuração e o banco permanecem em `/etc/tempo-agent` e
`/var/lib/tempo-agent`, para diagnóstico e reinstalação.

## Pacote transportável

```bash
make package-client
```

O tarball e seu SHA-256 são gravados em `dist/`. Após extrair o pacote em outra
máquina, execute `sudo ./scripts/install-pilot-components.sh` a partir da raiz
extraída.
