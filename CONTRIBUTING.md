# Como contribuir com o Compasso

Obrigado por considerar uma contribuição. O Compasso ainda está em fase de
piloto; mudanças pequenas, verificáveis e acompanhadas de contexto são as mais
fáceis de revisar.

## Preparar o ambiente

Instale as dependências descritas em [docs/development.md](docs/development.md)
e prepare o frontend:

```bash
npm ci --prefix admin-ui
make test
```

Configurações reais devem ser criadas a partir dos arquivos `*.example.toml` e
mantidas fora do Git. Use `var/` para bancos e estado de desenvolvimento.

## Antes de alterar uma interface

- `admin-ui/` é a única interface administrativa vigente;
- `local-ui/` contém as duas janelas vigentes do cliente Linux;
- imagens de referência, protótipos e notas privadas não são implementações do
  produto;
- uma mudança no servidor ou no agente não autoriza mudanças automáticas em
  todas as interfaces consumidoras.

Se uma solicitação puder se referir a mais de uma implementação, confirme o
escopo antes de editar. Preserve padrões visuais, acessibilidade e linguagem já
adotados.

## Regras técnicas

- adicione ou atualize testes para mudanças de comportamento;
- execute `make test` antes de abrir um pull request;
- não edite migrações já distribuídas: crie o próximo arquivo numerado;
- não registre tokens, senhas, cookies, bancos, chaves ou endereços privados;
- não versione `bin/`, `dist/`, `var/`, caches ou materiais de `.private/`;
- mantenha o agente funcional com a última autorização válida sem rede;
- documente mudanças de protocolo e compatibilidade entre agente e servidor.

## Pull requests

Descreva o problema, a solução, os riscos e como o resultado foi validado.
Separe refatorações amplas de mudanças funcionais quando possível. Alterações
que afetam bloqueio, autenticação, instalação, migrações ou persistência devem
incluir um plano de recuperação.

Use mensagens de commit objetivas, por exemplo:

```text
fix(agent): preserva estado após falha de sincronização
docs: esclarece instalação do painel administrativo
```

Vulnerabilidades não devem ser relatadas em issues públicas. Consulte
[SECURITY.md](SECURITY.md).

## Licença das contribuições

Ao enviar uma contribuição, você confirma que pode licenciá-la e concorda que
ela seja distribuída sob a mesma licença do projeto:
[`AGPL-3.0-or-later`](LICENSE).
