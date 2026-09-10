# Implantação

O diretório `deploy/` contém somente a implantação da interface administrativa.
A API usa o [`compose.yaml`](../compose.yaml) da raiz e é distribuída pelo pacote
`compasso-server`.

## Interface administrativa

`deploy/admin-ui/` fornece:

- `compose.yml`: Nginx para servir o build estático;
- `.env.example`: bind, porta e diretório do build;
- `default.conf`: fallback de SPA, healthcheck e cabeçalhos básicos;
- `runtime-config.js`: exemplo que aponta a interface para a API no mesmo host,
  porta `8181`.

Fluxo manual:

```bash
cd admin-ui
npm ci
npm run build
sudo install -d /srv/sites/compasso-admin-ui
sudo cp -a dist/. /srv/sites/compasso-admin-ui/
cd ../deploy/admin-ui
cp .env.example .env
docker compose up -d
```

Antes de expor o painel fora de uma rede confiável, configure HTTPS, cookies
seguros, origem administrativa, firewall e backup de acordo com o ambiente. O
Compasso não instala túnel, DNS nem certificado.

## Publicação por SSH

O script `scripts/publish-admin-ui.sh` automatiza build e cópia para um host já
preparado. Destino e URL pública da API devem ser informados pelo operador; o
diretório possui um padrão configurável. Nenhum endereço de implantação
pertence ao repositório.

Para não repetir opções, crie o arquivo ignorado
`.private/deploy/admin-ui.env` uma vez:

```text
COMPASSO_ADMIN_UI_DEPLOY_TARGET=usuario@servidor
COMPASSO_ADMIN_UI_DEPLOY_DIRECTORY=/srv/sites/compasso-admin-ui
COMPASSO_API_BASE_URL=https://api.exemplo.com
```

Depois, cada atualização exige somente:

```bash
make publish-admin-ui
```

Configurações específicas de máquinas, ferramentas auxiliares de inspeção e
composes pessoais devem ficar em `.private/deploy/`, que é ignorado pelo Git.
