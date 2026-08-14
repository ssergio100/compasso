# Proposta de reposicionamento familiar do Compasso

## Estado deste documento

Este documento registra uma proposta para discussão na próxima sessão. Ele não
representa ainda uma decisão definitiva e não autoriza, por si só, alterações
no banco de dados, nos serviços, nos pacotes ou na interface.

## Motivação

O objetivo original do projeto é ajudar responsáveis a controlar o tempo de
uso e proteger crianças que têm acesso a computadores.

A interface atual apresenta o sistema como um painel de administração de
computadores. Essa linguagem e sua identidade visual dão ao produto uma
aparência mais empresarial, embora empresas provavelmente escolham soluções
mais complexas e voltadas à gestão corporativa.

O Compasso oferece deliberadamente uma solução simples para uso familiar. Ele
foi pensado para estabelecer limites e rotinas cotidianas, e não para enfrentar
um usuário técnico determinado a contornar todas as proteções do sistema.

## Direção de produto proposta

A entidade principal percebida pelo responsável deve ser a **criança**, e não o
computador. O computador continua existindo como elemento técnico vinculado à
criança, mas não deve comandar a organização nem a linguagem das telas
principais.

Exemplos de mudança de linguagem:

| Atual | Direção proposta |
|---|---|
| Computadores | Crianças |
| Adicionar computador | Adicionar criança |
| Computador do quarto | Nome da criança, por exemplo, Ana |
| Agora | Hoje da Ana |
| Administração | Configurações |
| Estado técnico | Computador vinculado ou uma seção técnica secundária |

As ações continuam sendo aplicadas tecnicamente ao computador e à conta Linux
controlada. A nova linguagem deve aproximar o produto de seu público sem
esconder informações técnicas quando elas forem necessárias para instalação,
diagnóstico ou manutenção.

## Limitação do modelo atual

Sem alteração do banco de dados, cada criança apresentada na interface
continuará correspondendo tecnicamente a um dispositivo.

Esse modelo é coerente quando existe uma relação prática de **uma criança por
computador ou por conta controlada**. Se duas crianças utilizarem a mesma conta
no mesmo computador, o sistema não consegue manter limites, rotinas e consumo
separados para cada uma.

Essa limitação deve ser comunicada no guia de uso. Uma futura evolução para
várias crianças por computador exigiria uma mudança real no domínio e não
apenas uma troca de textos na interface.

## Nome sugerido

### Compasso Família

Mensagem sugerida:

> Tempo digital com cuidado.

O nome **Compasso** já comunica ritmo, orientação e equilíbrio. A extensão
**Família** direciona o produto ao público correto sem exigir imediatamente a
renomeação de pacotes, serviços, banco de dados, URLs e componentes técnicos.

O nome e a mensagem ainda precisam ser confirmados antes da criação da nova
identidade visual.

## Direção visual preliminar

A identidade deve ser:

- acolhedora e familiar, sem parecer infantilizada;
- simples, clara e segura;
- menos corporativa, sem perder a seriedade necessária a um controle parental;
- baseada em formas mais suaves, cores mais vivas e ilustrações discretas;
- organizada em torno da criança, de seu tempo disponível e de suas rotinas;
- cuidadosa para não transmitir vigilância excessiva, punição ou confronto.

O redesenho deve ser elaborado e aprovado como conceito visual antes de ser
implementado.

## Documento de utilização proposto

Após a confirmação do posicionamento, deve ser criado o arquivo
`GUIA_PARA_FAMILIAS.md`, escrito para responsáveis sem conhecimento técnico.

Conteúdo previsto:

1. O que o Compasso Família faz.
2. O que o sistema não pretende fazer.
3. Como cadastrar uma criança.
4. A relação entre criança, conta Linux e computador.
5. Como definir limites diários.
6. Como criar rotinas de bloqueio.
7. Como oferecer tempo adicional.
8. O significado das ações pausar, bloquear, revogar acesso e excluir.
9. O comportamento do sistema sem conexão com a Internet.
10. Limitações de segurança e o público para o qual a solução foi projetada.
11. Instalação e primeiros passos em linguagem acessível.

## Pontos para a próxima sessão

1. Confirmar ou substituir o nome **Compasso Família**.
2. Confirmar a mensagem **Tempo digital com cuidado**.
3. Decidir se a primeira versão continuará assumindo uma criança por
   computador ou conta controlada.
4. Definir a nova linguagem das telas e separar conceitos familiares de termos
   técnicos.
5. Criar e aprovar conceitos visuais para desktop e celular.
6. Planejar a transição sem renomear prematuramente serviços e artefatos
   técnicos.
7. Redigir o `GUIA_PARA_FAMILIAS.md` depois que nome, linguagem e fluxo forem
   aprovados.
