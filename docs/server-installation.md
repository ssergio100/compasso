# Compasso Server — instalação no Debian 13

O pacote Debian instala a API e a interface administrativa em dois contêineres
independentes, administrados pelo mesmo Docker Compose. Ele não instala nem
configura túnel, proxy reverso, VPN, DNS, certificado ou firewall.

## Instalação

Copie o `.deb` para o servidor e instale-o:

```bash
sudo apt install ./compasso-server_<versão>_all.deb
```

Revise `/etc/compasso-server/compasso.env` e execute:

```bash
sudo /opt/compasso-server/scripts/install-server.sh
```

O instalador:

1. verifica Docker Engine e Docker Compose e pede autorização antes de instalar
   uma dependência ausente;
2. cria os dados em `/srv/docker/volumes/compasso` e os backups em
   `/srv/docker/backups/compasso`;
3. constrói e inicia API e interface nas portas configuradas;
4. não solicita usuário, senha, domínio ou configuração de infraestrutura.

Por padrão, API e painel escutam em todas as interfaces do host nas portas
`8181` e `8182`. Em uma rede doméstica, o painel pode ser aberto em
`http://IP-DO-SERVIDOR:8182`, e o frontend encontra a API automaticamente no
mesmo IP, porta `8181`.

O arquivo `.env` permite restringir o bind a `127.0.0.1`, trocar portas e
configurar URLs HTTPS. Essas alterações pertencem à implantação escolhida pelo
usuário e não fazem parte da instalação do Compasso.

Depois da instalação, abra `http://IP-DO-SERVIDOR:8182`. Se o banco ainda não
possuir administrador, o painel exibe “Configurar o Compasso” para criar usuário
e senha. A configuração inicial é desativada permanentemente após a criação do
primeiro acesso. Faça essa etapa antes de publicar o servidor na Internet.

## Operação

```bash
sudo /opt/compasso-server/scripts/status-server.sh
sudo /opt/compasso-server/scripts/backup-server.sh
sudo /opt/compasso-server/scripts/update-server.sh
sudo /opt/compasso-server/scripts/restore-server-backup.sh /srv/docker/backups/compasso/compasso-server-DATA.tar.gz
```

Uma atualização preserva `.env` e o banco externo ao diretório do pacote. A
restauração exige confirmação textual e move os dados
anteriores para o diretório de backups antes de recuperar o arquivo escolhido.

## Fronteira dos componentes

- `compasso-api`: Go, API JSON e SQLite; não contém nem serve HTML.
- `compasso-admin-ui`: Nginx não-root com HTML, CSS e JavaScript; não contém o
  servidor nem acessa o banco.
- agente cliente: não faz parte deste Compose e continua sendo um serviço
  systemd nativo na máquina monitorada.
