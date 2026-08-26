# Proposta de direcionamento familiar do Compasso

## Estado deste documento

Este documento registra a direção de produto escolhida para orientar discussão,
desenho e planejamento. Ele não autoriza, por si só, alterações no banco de
dados, na API, no agente, nos serviços, nos pacotes ou nas interfaces existentes.

Antes de implementar uma nova interface, deverá ser definido expressamente o
papel da interface administrativa vigente e qual interface será considerada a
implementação oficial para o público familiar.

## Objetivo

Direcionar o Compasso prioritariamente às famílias, oferecendo a responsáveis
uma forma simples e amigável de estabelecer limites de tempo e rotinas para o
uso de computadores por crianças.

O direcionamento familiar pertence à identidade, à linguagem e à camada de
apresentação. O núcleo técnico continuará neutro e poderá ser utilizado em
outros contextos, inclusive organizacionais, sem que esses contextos determinem
a experiência do produto principal.

## Motivação

O objetivo original do projeto é ajudar responsáveis a controlar o tempo de uso
e proteger crianças que têm acesso a computadores.

Embora a tecnologia também possa ser utilizada em empresas e outros ambientes,
esses públicos normalmente dispõem de necessidades e soluções de administração
mais amplas. As famílias têm um propósito mais específico e tendem a valorizar
uma experiência acessível, direta e sem linguagem técnica desnecessária.

Por isso, o Compasso não precisa apresentar-se como uma ferramenta genérica
para todos os cenários. Ele pode manter um núcleo reutilizável e, ao mesmo
tempo, posicionar seu produto principal para o público que provavelmente obterá
mais valor de sua simplicidade.

## Princípio de arquitetura

O reposicionamento não deve transformar conceitos familiares em nomes técnicos
do sistema.

Continuam neutros e inalterados, salvo decisão técnica futura independente:

- o cadastro e o identificador do dispositivo;
- a conta Linux controlada pelo agente;
- o banco de dados e seus relacionamentos;
- os contratos e caminhos da API;
- os protocolos de sincronização;
- os nomes de serviços, executáveis e pacotes;
- os mecanismos de pareamento, autenticação, comandos e diagnóstico.

A interface familiar traduz esses elementos para a linguagem de responsáveis e
crianças, mas não altera nem esconde seu significado quando informações técnicas
forem necessárias.

## Entidade percebida na interface familiar

Nas funções cotidianas, a entidade principal percebida pelo responsável deve
ser a **criança**, representando a pessoa que utiliza a conta Linux controlada.
O computador deve aparecer como elemento técnico vinculado e secundário.

É importante distinguir os conceitos:

- **criança** é a pessoa apresentada na experiência familiar;
- **conta Linux controlada** é a conta configurada no agente e efetivamente
  submetida às regras;
- **computador** é o dispositivo no qual o agente e a conta estão configurados.

O termo correto é **conta Linux controlada**, e não simplesmente “conta
logada”. Outras contas podem iniciar sessão no mesmo computador sem serem o alvo
das regras configuradas para aquela instalação.

## Modelo suportado na primeira versão

Sem alteração do domínio técnico, cada criança exibida na interface familiar
continuará correspondendo a uma instalação do Compasso que controla uma única
conta Linux.

Essa decisão permite adequar a experiência sem modificar banco de dados, API ou
agente, mas impõe limitações que não devem ser escondidas:

- dois computadores usados pela mesma criança aparecerão como dois cadastros
  independentes;
- o consumo de tempo não será agregado entre computadores;
- duas crianças que utilizem a mesma conta Linux compartilharão limites,
  rotinas e consumo;
- o sistema não administrará separadamente várias contas controladas por uma
  única instalação;
- excluir o cadastro familiar continuará removendo o dispositivo técnico e os
  dados associados conforme o comportamento atual do servidor.

Uma futura relação entre crianças, vários dispositivos e várias contas exigirá
uma evolução real do domínio. Essa evolução não faz parte desta proposta de
apresentação.

## Camada de apresentação familiar

Poderá ser criada uma interface familiar própria, consumindo a mesma API e
preservando o núcleo existente. Essa interface será responsável por linguagem,
identidade visual, organização das informações e fluxos voltados às famílias.

A existência de uma interface familiar não justifica manter indefinidamente
duas interfaces completas com as mesmas responsabilidades. Antes da
implementação, a interface vigente deverá receber um papel explícito, por
exemplo:

- tornar-se a interface familiar oficial;
- permanecer como console técnico para instalação e diagnóstico; ou
- ser mantida apenas durante uma transição formalmente planejada.

Interfaces descontinuadas não deverão receber novas funcionalidades,
documentação ou correções do produto familiar.

## Linguagem proposta

Nas tarefas cotidianas, a linguagem deve referir-se à criança. Nas tarefas de
instalação, segurança e diagnóstico, deve referir-se ao computador, à conta ou
ao agente correspondente.

| Linguagem atual ou técnica | Interface familiar |
|---|---|
| Computadores | Crianças |
| Adicionar computador | Adicionar criança |
| Computador do quarto | Ana |
| Agora | Hoje da Ana |
| Administração | Configurações |
| Mais tempo para o computador | Mais tempo para Ana |
| Bloquear computador | Bloquear agora |
| Desbloquear computador | Permitir acesso novamente |
| Estado do computador | Computador vinculado |
| Token do dispositivo | Credenciais do computador, em área técnica |
| Revogar token | Desconectar computador, com explicação técnica |

Nem toda ocorrência de “computador” deve ser substituída. Estado online,
sincronização, agente, sessão gráfica, credenciais, comunicação e pareamento são
propriedades técnicas do equipamento e devem continuar identificadas como tal.

### Cuidado com a ação “Pausar”

No comportamento atual, pausar o monitoramento suspende restrições e
contabilização; não significa impedir o uso pela criança. Na interface familiar,
essa ação deve receber o nome inequívoco **Parar proteção**. Enquanto a proteção
estiver parada, o computador continuará disponível, mas limites, rotinas e
contabilização não serão aplicados. A ação inversa deverá chamar-se **Retomar
proteção**.

### Significado de “Bloquear”

A ação **Bloquear agora** interrompe o acesso à conta controlada apresentando a
tela de bloqueio do sistema. Ela não encerra a sessão, não fecha programas, não
descarta trabalhos abertos e não desliga o computador. A contabilização também
é interrompida enquanto o bloqueio manual estiver ativo.

Esse efeito deve ser explicado junto à confirmação da ação, em linguagem
semelhante a:

> A tela será bloqueada, mas os programas e trabalhos abertos continuarão como
> estão.

A ação inversa deverá chamar-se **Permitir acesso novamente**. Ela remove o
bloqueio imposto pelo Compasso, mas não digita a senha nem abre remotamente a
sessão. A criança ainda deverá desbloquear a tela normalmente com a senha da
conta.

**Bloquear agora** e **Parar proteção** produzem efeitos opostos sobre o acesso
e não podem compartilhar o mesmo rótulo ou explicação.

## Organização sugerida da experiência

### Funções familiares principais

- seleção da criança;
- tempo disponível e utilizado hoje;
- limites diários;
- rotinas;
- concessão de tempo adicional;
- bloqueio imediato;
- avisos e histórico apresentados em linguagem acessível.

### Informações técnicas secundárias

- computador vinculado;
- estado online e última sincronização;
- conta controlada, quando essa informação estiver disponível;
- estado do agente e da sessão gráfica;
- pareamento e credenciais;
- comunicação, diagnóstico e manutenção;
- exclusão ou desconexão do computador, com consequências explícitas.

O responsável deve conseguir realizar as tarefas familiares comuns sem
compreender identificadores, tokens, revisões de política ou detalhes de
sincronização.

## Nome e mensagem sugeridos

### Compasso Família

Mensagem sugerida:

> Tempo digital com cuidado.

O nome **Compasso** comunica ritmo, orientação e equilíbrio. A extensão
**Família** identifica o público do produto sem exigir a renomeação de pacotes,
serviços, banco de dados, URLs ou componentes técnicos.

O nome familiar deve ser entendido como identidade da experiência e do produto
principal, não como restrição do núcleo técnico. O nome e a mensagem ainda
precisam ser confirmados antes da criação da identidade visual.

Como complemento funcional ao slogan, a apresentação poderá explicar o produto
de forma direta:

> Limites de tempo e rotinas para os computadores Linux da família.

## Direção visual preliminar

A interface familiar deve ser:

- acolhedora e familiar, sem parecer infantilizada;
- simples, clara e segura;
- organizada em torno da criança, de seu tempo disponível e de suas rotinas;
- baseada em formas suaves, cores convidativas e ilustrações discretas;
- cuidadosa para não transmitir punição, confronto ou vigilância excessiva;
- capaz de manter informações técnicas acessíveis sem colocá-las no centro da
  experiência.

Uma eventual interface ou console técnico poderá preservar uma apresentação
mais neutra. O conceito visual familiar deverá ser elaborado, avaliado e
aprovado antes da implementação.

## Escopo desta proposta

### Incluído

- posicionamento prioritário para famílias;
- identidade e linguagem familiares;
- organização da experiência em torno da criança;
- criação ou adaptação de uma camada de apresentação familiar;
- separação visual entre tarefas cotidianas e informações técnicas;
- documentação de utilização para responsáveis.

### Não incluído

- alteração do banco de dados ou da API;
- criação de uma entidade técnica de criança;
- agregação de uso entre vários computadores;
- suporte a várias crianças na mesma conta controlada;
- suporte a várias contas controladas por uma única instalação;
- renomeação de serviços, executáveis, pacotes ou protocolos;
- desenvolvimento de funcionalidades específicas para empresas;
- replicação automática das mudanças em interfaces legadas, protótipos ou
  implementações descontinuadas.

## Critérios para avaliar a interface familiar

A proposta será considerada coerente quando um responsável sem conhecimento
técnico conseguir:

1. cadastrar uma criança e compreender que será necessário vincular um
   computador;
2. identificar rapidamente o tempo disponível e utilizado;
3. configurar limites e rotinas sem lidar com termos internos;
4. conceder tempo adicional e bloquear o acesso com segurança;
5. entender claramente o efeito de parar a proteção e a diferença em relação
   ao bloqueio da tela;
6. localizar informações técnicas quando precisar instalar ou diagnosticar;
7. compreender as consequências de desconectar ou excluir um cadastro.

## Guia de utilização proposto

Depois da confirmação do nome, da linguagem e dos fluxos, deverá ser criado o
arquivo `GUIA_PARA_FAMILIAS.md`, escrito para responsáveis sem conhecimento
técnico.

Conteúdo previsto:

1. O que o Compasso Família faz.
2. O que o sistema não pretende fazer.
3. A relação entre criança, conta Linux controlada e computador.
4. Como cadastrar uma criança e vincular o computador.
5. Como definir limites diários.
6. Como criar rotinas de bloqueio.
7. Como oferecer tempo adicional.
8. A diferença entre bloquear, permitir acesso novamente e parar a proteção.
9. Como desconectar ou excluir um computador.
10. O comportamento do sistema sem conexão com a Internet.
11. Limitações de segurança e o público para o qual a solução foi projetada.
12. Instalação e primeiros passos em linguagem acessível.

## Decisões pendentes antes da implementação

1. Confirmar ou substituir o nome **Compasso Família**.
2. Confirmar a mensagem **Tempo digital com cuidado**.
3. Definir o papel da interface administrativa vigente.
4. Confirmar qual implementação será a interface familiar oficial.
5. Confirmar o vocabulário das ações **Bloquear agora**, **Permitir acesso
   novamente**, **Parar proteção** e **Retomar proteção**, e fechar os termos
   para desconectar e excluir.
6. Definir como o nome da criança e a identificação técnica do computador serão
   apresentados usando o modelo atual.
7. Criar e aprovar conceitos visuais para desktop e celular.
8. Validar os principais fluxos com responsáveis sem conhecimento técnico.
9. Planejar a transição sem renomear prematuramente artefatos técnicos.
10. Redigir o `GUIA_PARA_FAMILIAS.md` após a aprovação da experiência.
