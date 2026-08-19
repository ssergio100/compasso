# Diretrizes obrigatórias do repositório

## Escopo das interfaces

- Antes de alterar qualquer interface, identifique qual implementação está
  vigente para a solicitação.
- Se houver mais de uma implementação possível, pare e pergunte ao usuário
  qual deve ser alterada.
- Nunca replique uma correção em protótipos, interfaces legadas ou
  descontinuadas sem autorização explícita.
- Não gaste implementação, documentação ou testes com uma interface
  descontinuada. Corrija e valide somente a interface vigente confirmada.
- Uma mudança no servidor ou no agente não autoriza automaticamente mudanças
  em todas as interfaces consumidoras.
