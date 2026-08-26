# Interface administrativa Compasso

Esta é a única interface administrativa vigente do Compasso. O conceito visual
é o “painel de ritmo”, com foco em leitura simples por pessoas sem conhecimento
técnico.

```bash
npm install
npm run dev
```

Sem configuração, usa dados locais demonstrativos. Para consumir a API real, defina `VITE_COMPASSO_REMOTE=true` e `VITE_COMPASSO_API_BASE_URL`, ou use `COMPASSO_DEV_API_TARGET` como proxy de desenvolvimento.

O arquivo `public/runtime-config.js` é copiado para o build e, por padrão,
direciona a interface para a API no mesmo IP, porta `8181`. O bundle repete essa
resolução automática como proteção caso o arquivo esteja ausente, vazio ou
ainda utilize a grafia legada `apiBaseURL`. Uma implantação externa pode
substituir `apiBaseUrl` pela URL pública correspondente.

No modo remoto, tempo adicional permanece em **Sincronizando** sem alterar o
contador localmente. O saldo é recarregado somente quando o agente reconhece o
identificador da operação retornado pela API.

A página **Comunicação** acompanha os intercâmbios entre agente, API e
interface. Ela consulta somente registros novos a cada segundo, permite busca,
filtros e inspeção de metadados sanitizados. A retenção é configurável entre 1
e 90 dias na própria página; o servidor usa 30 dias por padrão e aceita até 365
dias pela API. A exclusão manual afeta somente os logs do computador
selecionado, sem remover histórico de uso, bônus ou configurações.

```bash
npm run typecheck
npm run build
```

## Princípio de revisão

Preserve a coerência visual e a usabilidade do produto como um sistema. Quando
uma solicitação conflitar com padrões já estabelecidos, introduzir ambiguidade
ou prejudicar a experiência, explique o conflito e proponha uma alternativa
antes de implementar. Não aplique silenciosamente uma decisão ruim apenas por
ter sido solicitada.
