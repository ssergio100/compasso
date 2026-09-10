# Demonstração e capturas de tela

O painel vigente possui dados demonstrativos incorporados. Esse modo não usa
API, não pede credenciais e não persiste alterações.

## Iniciar

```bash
cd admin-ui
npm ci
npm run dev
```

Abra:

```text
http://127.0.0.1:4175/?preview=visuals
```

O parâmetro `preview=visuals` é obrigatório porque a configuração padrão do
painel procura uma API real na porta `8181`.

## Roteiro curto

Para uma apresentação consistente, capture ao menos:

1. **Agora**, mostrando saldo, estado e ações principais;
2. **Limites**, mostrando as cotas da semana;
3. **Rotinas**, mostrando a agenda familiar;
4. **Atividade**, mostrando uma operação concluída e suas etapas;
5. uma das telas em largura de celular.

Use somente os dados fictícios incluídos no modo demonstrativo. Não faça
capturas de um ambiente real com nomes, endereços, tokens ou histórico de uma
família.

## Verificação antes das imagens

```bash
npm run typecheck
npm run build
```

Confira também o console do navegador. Uma captura de divulgação deve ser feita
com o build atual, sem erros, overlays de desenvolvimento ou extensões visíveis.
