# Plano de adequação — cliente Linux instalável

> **Registro histórico:** este plano acompanha os pilotos que usavam logout por
> D-Bus. A implementação atual usa `loginctl lock-session`; consulte
> `agent/README.md` para o comportamento vigente.

## Objetivo

Entregar um cliente que uma pessoa comum consiga instalar sem compilador,
terminal ou edição manual de arquivos. O gerenciador de pacotes deve apresentar
as dependências ausentes, pedir autorização administrativa e instalá-las.

O agente será instalado nativamente e iniciado pelo `systemd` na máquina
monitorada. Docker não será instalado, solicitado nem utilizado para executar o
agente. Em produção, Docker pertence somente ao servidor.

O primeiro pacote suportará Debian, Ubuntu, Zorin, Mint e derivados em `amd64`,
com systemd e logind. A integração gráfica não consulta o nome do ambiente: um
helper descobre capacidades no D-Bus e usa adaptadores extensíveis. A separação
entre núcleo, interface e sessão deve permitir adicionar novos adaptadores,
pacotes RPM e outras arquiteturas sem reescrever o agente.

## Estratégia de baixo custo

- manter o agente, protocolo, banco, servidor e motor de política atuais;
- manter temporariamente a interface Python/GTK, declarando suas dependências
  no pacote em vez de presumir que já estejam instaladas;
- gerar o binário na máquina de desenvolvimento em um contêiner de build com o
  Go oficial, nunca com `gccgo`; o contêiner é descartável e não acompanha o
  pacote instalado no cliente;
- criar primeiro um `.deb`; RPM, repositório assinado e substituição do GTK
  ficam fora da adequação imediata;
- manter o gate PAM desativado; a autenticação é permitida e o agente aplica
  logout seguro somente depois que a sessão gráfica estiver estabelecida.

## Etapa 1 — binário portátil

- [x] Criar build reproduzível em Docker usando o compilador Go oficial, com a
  imagem Go 1.26 sobre Debian 12 fixada por digest.
- [x] Confirmar que o pacote do cliente não contém configuração de contêiner e
  não declara Docker como dependência.
- [x] Gerar binários sem dependência de `libgo.so`.
- [x] Remover símbolos de depuração dos artefatos distribuídos.
- [x] Fazer o teste de empacotamento falhar se `ldd` encontrar `libgo`.
- [ ] Confirmar execução em Zorin e Debian 13 sem Go instalado.

## Etapa 2 — pacote Debian

- [x] Gerar `compasso-client_<versão>_amd64.deb`.
- [x] Declarar dependências runtime no metadado do pacote.
- [x] Instalar binários, unidade systemd, política D-Bus e atalhos do desktop
  em caminhos padronizados.
- [ ] Tornar instalação, atualização, remoção e repetição idempotentes.
- [x] Preservar configuração e estado em `remove`; apagá-los somente em
  `purge` explicitamente solicitado.
- [x] Habilitar o agente inicialmente sem gate PAM.

Validações automatizadas já executadas no pacote piloto:

- [x] Metadados e scripts mantenedores possuem formato válido.
- [x] Configuração é instalada com modo `0600`.
- [x] Binários extraídos executam sem `libgo` ou biblioteca ausente.
- [x] O agente valida a configuração extraída do pacote.
- [x] O SHA-256 do artefato confere.
- [x] Nome **Compasso — Adicionar tempo** aplicado ao menu, à janela e aos
  metadados AppStream do `pilot3`.
- [x] Pacote `pilot4` contém assistente gráfico, autostart de primeira execução,
  helper privilegiado, política Polkit e dependência declarada.
- [x] `pilot4` passou nos testes Go/Python, validação AppStream, inspeção do
  `.deb` e verificação de bibliotecas runtime.
- [x] `pilot5` inicia o seletor sem conta escolhida e exige confirmação nominal
  antes de permitir a configuração (teste automatizado da lógica da interface).
- [x] `pilot5` passou pela suíte completa e pela validação estrutural do `.deb`.
- [x] Instalação do `pilot5` tentada no Debian 13; o Discover informou falha ao
  resolver dependências porque `policykit-1` não existe como pacote binário
  nessa versão.
- [x] APT do Debian 13 limpo resolveu `pkexec` e todas as demais dependências e
  instalou o `pilot6` em contêiner descartável.
- [x] Instalação real do `pilot6` no Debian 13 KDE executada: as dependências
  foram resolvidas, mas o pacote reutilizou credenciais antigas, iniciou o
  agente sem nova confirmação e o encerramento abrupto deixou o Plasma em tela
  preta. O artefato foi invalidado.
- O teste real do `pilot7` foi substituído pelo `pilot8` antes da execução.
- [x] APT do Debian 13 limpo instalou o `pilot7` em contêiner descartável sem
  criar o marcador de configuração nem habilitar o serviço.
- [x] `pilot8` aguarda a primeira sincronização concluída depois de um novo
  login sem saldo antes de decidir pelo logout (testes automatizados).
- O `pilot8` não foi instalado: foi substituído antes do ensaio pelo `pilot9`,
  que acrescenta logout normal, saldo confirmado e presença gráfica explícita.
- [x] Configurar e iniciar o `pilot9` no Debian 13 KDE revelou incompatibilidade
  entre a leitura de `/proc/sys/kernel/random/boot_id` e o hardening
  `ProcSubset=pid`; o serviço falhou antes da sincronização e o artefato foi
  invalidado.
- [x] O ensaio do `pilot10` confirmou que o agente iniciava e alcançava o
  servidor, mas revelou que uma reconfiguração com o serviço já ativo gravava
  o token novo sem reiniciar o processo. O agente continuava enviando o token
  anterior e recebendo HTTP 401; o artefato foi invalidado.
- [x] Ensaiar o `pilot11` completo no Debian 13 KDE real: configuração,
  confirmação online, contagem e alertas passaram, mas o logout falhou de forma
  segura porque o serviço D-Bus do Plasma ainda não tinha sido ativado. O
  artefato foi invalidado.
- [x] Gerar e validar sem instalação o
  `compasso-client_0.1.0~pilot11_amd64.deb`.
- [x] Instalar e configurar o `pilot11` no Debian 13 KDE pela interface; a
  janela aguardou a comunicação e o agente apareceu online no servidor (teste
  real).
- [x] Validar o `pilot12` no Debian 13 KDE com o serviço de logout inicialmente
  inativo: ao expirar a cota, o agente ativou o provedor e realizou o logout
  (teste real).
- [x] Confirmar retorno ao SDDM sem tela preta após o logout executado pelo
  `pilot12` (teste real).
- [x] Confirmar novo login normal depois do logout executado pelo `pilot12`; o
  saldo extra sincronizou, os alertas apareceram e um segundo logout terminou
  normalmente (teste real).
- [x] Confirmar logout seguro durante rotina ativa mesmo com tempo extra
  disponível (teste real do `pilot12` com a rotina **Dormir**).
- [x] Gerar e validar sem instalação o
  `compasso-client_0.1.0~pilot12_amd64.deb`.

- [x] Instalar o `pilot9` no Debian 13 KDE sem iniciar automaticamente o agente
  nem reutilizar as credenciais da instalação anterior.

- [x] `pilot9` gerado com `compasso-session-logout`, âncora de saldo e
  identidade de sessão por boot; estrutura do `.deb`, dependências, AppStream,
  configuração e bibliotecas runtime validadas sem instalação.
- [x] Remover a leitura incompatível de `/proc/sys` e manter a identidade da
  sessão em `RuntimeDirectory`, preservada nos reinícios automáticos e renovada
  após reboot ou parada explícita (teste automatizado).
- [x] Gerar e validar sem instalação o
  `compasso-client_0.1.0~pilot10_amd64.deb`, incluindo binários portáteis,
  dependências, AppStream e unidade systemd endurecida.

Dependências iniciais previstas:

```text
systemd
dbus
libpam-runtime
ca-certificates
python3
python3-gi
gir1.2-gtk-4.0
libnotify-bin
pkexec
```

## Etapa 3 — configuração sem terminal

- [x] Adicionar ao menu a aplicação **Compasso — Configurar agente**.
- [x] Solicitar conta controlada, URL do servidor, `device_id` e
  `device_token` em uma janela.
- [x] Preencher por padrão `https://apicompasso.smresume.com`.
- [x] Gravar credenciais sem colocá-las em argumentos de processo ou logs.
- [x] Manter o serviço desabilitado enquanto a configuração inicial estiver
  incompleta e iniciá-lo somente depois da validação.
- [x] Abrir automaticamente o assistente no próximo início de sessão enquanto
  a configuração não tiver sido concluída.
- [x] Reiniciar o processo depois de cada configuração, esperar um heartbeat
  aceito e somente então exibir **Servidor online** e criar o marcador de
  configuração (testes automatizados).
- [x] Pré-selecionar de verdade a primeira conta comum exibida e habilitar o
  checkbox de confirmação correspondente (teste automatizado da seleção).
- [ ] Planejar como melhoria posterior o pareamento por código curto e uso
  único, sem bloquear a entrega inicial do `.deb`.

## Etapa 4 — logout seguro independente do desktop

- [x] Manter a aplicação da política independente do nome do display manager e
  do ambiente gráfico (teste automatizado da delegação neutra).
- [x] Debian 13 KDE: introspecção inofensiva confirmou que a sessão Plasma
  publica `org.kde.Shutdown.logout` no D-Bus do usuário.
- [x] Debian 13: a ponte genérica do systemd alcançou o D-Bus, apresentou os
  métodos disponíveis e terminou em 21 ms com código zero; o ensaio usou
  timeout de 15 segundos e não encerrou a sessão.
- [x] Detectar pelo logind que a sessão gráfica ficou `active` e aguardar dez
  segundos de estabilização antes de solicitar logout após relogin bloqueado.
- [x] Implementar helper neutro que descobre capacidades no D-Bus da sessão e
  usa adaptadores de logout normal, sem inspecionar o nome do desktop e sem
  fallback de encerramento abrupto (testes automatizados).
- [x] Debian 13 KDE: executar o helper portátil em modo `-probe` pela ponte do
  agente; ele descobriu `plasma-session`, terminou em 70 ms com código zero e
  não alterou a sessão (teste real).
- [x] Se nenhum adaptador estiver disponível ou a chamada falhar, manter a
  sessão aberta e nunca recorrer ao encerramento abrupto (teste automatizado).
- [x] Debian 13 KDE/SDDM: o adaptador concluiu o logout e retornou ao greeter;
  o login seguinte abriu o desktop normalmente, sem tela preta (teste real).
- [ ] Não ativar gate PAM: a autenticação permanece permitida por requisito.
- [ ] Validar notificações e interface local em GNOME e KDE.

## Etapa 5 — testes reais

- [x] Debian 13 KDE: primeira tentativa de abrir o `pilot1` executada; o
  Discover encerrou com `SIGSEGV` em Qt/QML antes da instalação. O pacote foi
  aceito na simulação do APT e recebeu metadados AppStream no `pilot2` para nova
  tentativa.
- [x] Debian 13 KDE: pacote `pilot3` instalado e validado pela interface
  gráfica, com o nome **Compasso — Adicionar tempo**.
- [ ] Zorin: instalar o `.deb` por interface gráfica em sistema já configurado.
- [ ] Zorin: atualizar e remover sem perder login ou estado indevidamente.
- [x] Debian 13 KDE limpo: abrir o `.deb`, autorizar dependências e instalar
  sem terminal.
- [ ] Debian 13 KDE: configurar pelo menu e confirmar agente online.
- [ ] Debian 13 KDE: validar alerta, bônus local e operação offline.
- [ ] Debian 13 KDE: relogar bloqueado e o agente acionar automaticamente o
  logout; o helper isolado já retornou ao SDDM sem tela preta.
- [ ] Repetir instalação, atualização e remoção para verificar idempotência.

## Critério de pronto desta adequação

- [ ] Uma pessoa baixa e abre um único `.deb`.
- [ ] O instalador gráfico mostra e instala dependências após autorização.
- [x] Nenhum compilador ou ferramenta Go é exigido no cliente.
- [ ] Toda configuração normal é concluída pela interface gráfica.
- [ ] O agente fica online e inicia automaticamente após reboot.
- [ ] O pacote funciona em Zorin/GDM e Debian 13 KDE/SDDM.
- [ ] A remoção nunca deixa o login gráfico inutilizável.

## Fora do escopo imediato

- pacote RPM para Fedora/openSUSE;
- AUR/Arch e distribuições sem systemd/logind;
- repositório APT assinado e atualizações automáticas;
- substituição da interface GTK por interface local web;
- pareamento por QR code.

Esses itens permanecem possíveis sem alterar o núcleo do agente, mas não devem
atrasar o piloto em Debian e Zorin.

O desacoplamento do painel administrativo é uma frente independente, descrita
em `docs/admin-frontend-decoupling-plan.md`, e não altera o formato do pacote
cliente.
