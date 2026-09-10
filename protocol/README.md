# Protocolo

`protocol/v1` contém os tipos Go compartilhados pelo agente e pelo servidor no
heartbeat. A versão da pasta representa o contrato da API e é independente da
versão do binário em `VERSION`.

O cliente anuncia capacidades pelo cabeçalho `X-Compasso-Protocol-Version`.
Mudanças compatíveis devem manter campos opcionais e valores padrão seguros.
Uma mudança incompatível exige novo contrato versionado e uma estratégia
explícita de implantação entre agente e servidor.

As regras completas de entrega, idempotência, revisões e confirmações estão em
[`docs/arquitetura-comunicacao.md`](../docs/arquitetura-comunicacao.md).

Execute os testes de ambos os lados após qualquer alteração:

```bash
go test ./protocol/... ./agent/syncclient/... ./server/...
```
