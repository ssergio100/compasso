# Compasso Rhythm

Uma exploração independente da interface administrativa do Compasso, criada do zero a partir do conceito visual “painel de ritmo”. O projeto não altera nem depende do protótipo `admin-ui-mobile`.

```bash
npm install
npm run dev
```

Sem configuração, usa dados locais demonstrativos. Para consumir a API real, defina `VITE_COMPASSO_REMOTE=true` e `VITE_COMPASSO_API_BASE_URL`, ou use `COMPASSO_DEV_API_TARGET` como proxy de desenvolvimento.

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
