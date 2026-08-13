# Compasso — Documento de Melhoria
## Bloqueio de Sessão, Comandos Remotos e Princípios de Correção

**Projeto:** Compasso
**Objetivo deste documento:** orientar futuras alterações no código, inclusive quando o projeto for trabalhado via Codex, preservando as decisões arquiteturais já tomadas e evitando correções por remendo.

---

# 1. Princípio principal de desenvolvimento

Ao corrigir um problema no Compasso, a prioridade deve ser **corrigir a causa**, e não contornar o sintoma.

Não se deve assumir que uma correção necessariamente exige mais código.

Em muitos casos, uma boa correção pode significar:

- remover uma condição incorreta;
- substituir uma abstração inadequada;
- eliminar um fluxo redundante;
- simplificar estados;
- centralizar uma decisão já existente;
- apagar código que deixou de fazer sentido.

Uma correção que apenas acrescenta novas camadas sobre uma lógica defeituosa tende a produzir:

- código morto;
- estados contraditórios;
- duplicação de regras;
- manutenção mais difícil;
- novos bugs;
- aumento desnecessário do consumo de tokens em futuras sessões de desenvolvimento.

**Diretriz:** antes de adicionar código, verificar se a solução correta é alterar, simplificar ou remover código existente.

---

# 2. Economia de tokens e eficiência

Todo o desenvolvimento do Compasso deve considerar que a interação com agentes de código ocorre dentro de um orçamento de tokens.

Por isso:

- evitar reescrever arquivos grandes quando uma alteração pequena for suficiente;
- evitar criar abstrações sem necessidade concreta;
- evitar duplicar regras já existentes;
- evitar comentários excessivos que apenas repetem o código;
- evitar implementar caminhos alternativos que não serão utilizados;
- preferir funções pequenas e reutilização de lógica existente;
- antes de modificar um arquivo, localizar exatamente onde a decisão relevante acontece;
- alterar o menor conjunto possível de arquivos;
- remover código obsoleto quando uma nova arquitetura tornar o fluxo antigo desnecessário.

**Objetivo:** máxima correção com mínima complexidade.

---

# 3. Problema identificado no mecanismo atual

O Compasso atualmente utiliza logout como mecanismo de bloqueio em determinados cenários.

Esse comportamento apresenta uma falha importante.

Exemplo:

1. Usuário está editando um documento.
2. Existem alterações ainda não salvas.
3. Compasso tenta encerrar a sessão.
4. O aplicativo impede ou interrompe o logout para perguntar se o documento deve ser salvo.
5. A sessão permanece ativa.
6. O usuário continua podendo utilizar a máquina.

Isso pode ocorrer tanto:

- quando a cota diária termina;
- quando uma rotina exige bloqueio;
- quando o servidor envia um comando remoto de bloqueio.

Existe ainda um segundo problema.

Se o servidor envia uma ordem de bloqueio e a primeira tentativa falha, o sistema pode considerar a ordem consumida/aplicada, mesmo que a sessão continue acessível.

Resultado possível:

```text
Servidor: máquina bloqueada
Cliente: máquina desbloqueada
```

Esse estado é inválido.

---

# 4. Nova estratégia: bloquear a sessão, não encerrar a sessão

O Compasso deve abandonar o logout como mecanismo normal de bloqueio.

A nova estratégia é:

```text
necessidade de bloqueio
        ↓
bloquear sessão gráfica
        ↓
manter programas e processos em execução
        ↓
preservar documentos não salvos
```

O objetivo é impedir o uso da interface sem destruir o estado da sessão.

Isso permite preservar:

- documentos abertos;
- editores;
- navegador;
- jogos;
- IDEs;
- terminais;
- aplicativos em geral.

O usuário poderá chegar à tela de desbloqueio e inclusive digitar sua senha.

Isso não é considerado uma falha.

Se o bloqueio ainda for válido, o agente poderá bloquear novamente a sessão quando detectar que ela voltou a ficar acessível.

---

# 5. Compatibilidade Linux

A implementação não deve depender de um único ambiente gráfico.

Deve existir uma camada pequena de abstração para bloqueio de sessão.

Exemplo conceitual:

```go
type SessionLocker interface {
    Lock() error
    IsLocked() (bool, error)
}
```

A primeira tentativa deve utilizar mecanismos genéricos do sistema, preferencialmente `systemd-logind` / `loginctl`, quando disponíveis.

Exemplo:

```bash
loginctl lock-session <sessao>
```

Quando necessário, podem existir backends específicos.

Prioridade inicial de suporte:

1. GNOME
2. KDE Plasma
3. Cinnamon
4. Xfce

Outros ambientes podem ser adicionados posteriormente.

**Não implementar vários backends antecipadamente se o backend genérico já resolver o ambiente em uso.**

A abstração deve existir para permitir extensão, não para obrigar a criação de código desnecessário.

---

# 6. Separação fundamental: política e comandos remotos

O Compasso possui duas naturezas diferentes de decisão.

## 6.1 Política

A política é sincronizada com o servidor e pode ser utilizada localmente quando o servidor estiver indisponível.

Exemplos:

- cota diária;
- rotinas;
- horários de bloqueio;
- tempo extra;
- alertas;
- regras de uso.

A política possui revisão própria.

Exemplo:

```text
policy_revision = 18
```

O agente pode manter localmente a última política válida.

---

## 6.2 Comandos remotos

Comandos remotos são ordens atuais do servidor e **não fazem parte da política local**.

Principais comandos:

- BLOCK
- PAUSE

Esses comandos não devem ser persistidos como autoridade offline.

Eles só são válidos enquanto o agente consegue consultar o servidor e confirmar seu estado atual.

---

# 7. Regra de prioridade

A prioridade geral é:

```text
Servidor acessível
        ↓
consultar estado atual no servidor
        ↓
aplicar comandos remotos vigentes
        ↓
avaliar política quando aplicável
```

Se o servidor estiver inacessível:

```text
ignorar comandos remotos anteriores
        ↓
utilizar somente a última política local válida
```

Essa regra é obrigatória.

---

# 8. Semântica do comando BLOCK

BLOCK é um comando remoto.

Quando o servidor estiver acessível e informar BLOCK ativo:

```text
Servidor → BLOCK
Agente   → bloqueia sessão
```

Se o usuário desbloquear a sessão com sua senha:

```text
sessão volta a ficar acessível
        ↓
agente consulta novamente o servidor
        ↓
BLOCK continua ativo
        ↓
agente bloqueia novamente
```

A ordem não precisa gerar uma nova revisão a cada tentativa.

O comando representa a intenção atual do servidor.

---

## 8.1 BLOCK sem comunicação com o servidor

Se o servidor ficar indisponível:

```text
último comando remoto = BLOCK
servidor inacessível
```

O agente **não deve continuar bloqueando apenas porque o último comando conhecido era BLOCK**.

O comando remoto deixa de ser autoridade.

O agente passa a decidir exclusivamente pela última política local válida.

Exemplo:

```text
Servidor indisponível
        ↓
ignorar BLOCK remoto anterior
        ↓
avaliar política local
        ↓
rotina exige bloqueio? → bloquear
cota acabou?           → bloquear
nenhuma regra?         → manter liberado
```

---

# 9. Semântica do comando PAUSE

PAUSE também é um comando remoto.

Enquanto PAUSE estiver ativo e o servidor estiver acessível:

- não contabilizar tempo da cota diária;
- não aplicar bloqueios normais derivados de rotina;
- não aplicar bloqueio por cota esgotada;
- continuar realizando comunicação com o servidor.

Exemplo:

```text
cota total: 120 minutos
uso atual: 83 minutos
PAUSE ativo
```

Mesmo após uma hora:

```text
uso atual continua: 83 minutos
```

Quando o servidor informar que PAUSE terminou, o agente volta imediatamente a avaliar a política.

---

## 9.1 PAUSE sem comunicação com o servidor

Se o servidor ficar indisponível:

```text
último comando remoto = PAUSE
servidor inacessível
```

O agente **não deve permanecer pausado**.

A pausa remota deixa de valer.

O agente volta a:

- contabilizar cota;
- aplicar rotinas;
- aplicar bloqueios definidos pela política local.

Portanto:

```text
sem servidor:
BLOCK remoto anterior → não vale
PAUSE remoto anterior → não vale
```

---

# 10. Regra conceitual

A arquitetura deve manter esta distinção:

> **Política fornece autonomia offline.
> Comandos remotos exigem autoridade atual do servidor.**

Nunca converter automaticamente um comando remoto em política local.

---

# 11. Revisões

Não utilizar `policy_revision` para representar comandos remotos.

Política e comandos possuem naturezas diferentes.

Preferencialmente:

```text
policy_revision
control_revision
```

ou estrutura equivalente.

Exemplo:

```text
policy_revision  = 18
control_revision = 42
```

Essas revisões não têm relação entre si.

`policy_revision` muda quando a política muda.

`control_revision` muda quando a intenção remota muda.

Exemplo:

```text
42 → BLOCK
43 → nenhuma ordem de bloqueio
44 → PAUSE
45 → PAUSE removido
```

Evitar gerar nova revisão simplesmente porque o agente tentou novamente executar o mesmo estado.

---

# 12. Receber não significa aplicar

O protocolo nunca deve assumir:

```text
comando recebido == comando executado
```

Deve existir distinção entre:

```text
RECEIVED
PENDING / APPLYING
APPLIED
```

ou semântica equivalente.

Exemplo:

```text
Servidor:
BLOCK solicitado

Agente:
comando recebido

Agente:
tentativa de bloqueio

Agente:
confirma sessão bloqueada

Servidor:
máquina efetivamente bloqueada
```

O servidor não deve mostrar a máquina como bloqueada apenas porque enviou a ordem.

---

# 13. Estado desejado e estado real

Sempre que útil, separar:

```text
desired_state
actual_state
```

Exemplo:

```text
desired_state = BLOCKED
actual_state  = UNLOCKED
```

Isso significa que ainda existe trabalho a realizar.

Somente quando:

```text
desired_state = BLOCKED
actual_state  = BLOCKED
```

o bloqueio está efetivamente aplicado.

Porém, para comandos remotos, `desired_state` só pode ser considerado válido enquanto houver comunicação atual com o servidor.

Não persistir esse estado como autoridade offline.

---

# 14. Estado exibido no servidor

A interface administrativa deve refletir a realidade reportada pelo agente.

Exemplo de estados úteis:

```text
Desbloqueado
Bloqueio solicitado
Bloqueado
Pausa ativa
Indisponível / offline
```

Não apresentar "Bloqueado" apenas porque o servidor enviou BLOCK.

O status deve depender da confirmação do agente.

---

# 15. Fluxo consolidado do agente

Fluxo conceitual:

```text
INÍCIO DO CICLO
        ↓
servidor está acessível?
        │
        ├── NÃO
        │     ↓
        │   carregar última política local válida
        │     ↓
        │   ignorar BLOCK remoto antigo
        │   ignorar PAUSE remoto antigo
        │     ↓
        │   avaliar cota + rotinas + regras
        │     ↓
        │   aplicar resultado
        │
        └── SIM
              ↓
            consultar política
              ↓
            atualizar política local se revisão mudou
              ↓
            consultar comandos remotos atuais
              ↓
            PAUSE ativo?
              │
              ├── SIM → suspender vigilância e contagem
              │
              └── NÃO
                    ↓
                  BLOCK ativo?
                    │
                    ├── SIM → bloquear sessão
                    │
                    └── NÃO → avaliar política
```

---

# 16. Ao desbloquear a sessão

O agente deve observar quando uma sessão que deveria estar bloqueada volta a ficar acessível.

Não presumir automaticamente que isso representa uma violação permanente.

Fluxo:

```text
sessão desbloqueada
        ↓
consultar servidor
        ↓
BLOCK remoto continua vigente?
        │
        ├── SIM → bloquear novamente
        │
        └── NÃO → avaliar política
```

Se o servidor estiver inacessível:

```text
não reutilizar BLOCK remoto antigo
        ↓
avaliar política local
```

---

# 17. Fim da cota e rotinas

Bloqueios derivados da política são diferentes do BLOCK remoto.

Exemplos:

```text
DAILY_QUOTA
ROUTINE
```

Esses bloqueios podem continuar sendo aplicados offline porque fazem parte da política local válida.

Exemplo:

```text
servidor indisponível
cota local esgotada
        ↓
bloquear sessão
```

Isso é correto.

Já:

```text
servidor indisponível
último BLOCK remoto conhecido
        ↓
não bloquear por esse motivo
```

---

# 18. Não misturar origem e efeito

Várias causas podem produzir o mesmo efeito:

```text
BLOCK remoto
cota esgotada
rotina
```

Todas podem resultar em:

```text
sessão bloqueada
```

Mas a origem deve continuar conhecida.

Exemplo:

```text
actual_state = BLOCKED
block_reason = REMOTE_COMMAND
```

ou:

```text
actual_state = BLOCKED
block_reason = DAILY_QUOTA
```

ou:

```text
actual_state = BLOCKED
block_reason = ROUTINE
```

Isso facilita diagnóstico e interface administrativa sem duplicar a lógica de bloqueio.

---

# 19. Diretriz para implementação no código existente

Antes de alterar qualquer coisa:

1. localizar o fluxo atual de logout;
2. localizar onde a aplicação decide que uma ordem remota foi aplicada;
3. localizar como `policy_revision` funciona hoje;
4. verificar se BLOCK atualmente gera ou não `policy_revision`;
5. localizar persistência local de política;
6. localizar heartbeat / sincronização de estado;
7. localizar como o servidor define o status visual do dispositivo.

Só depois disso modificar.

**Não pressupor a arquitetura atual. Ler o código primeiro.**

---

# 20. Estratégia de alteração

A implementação deve preferir a menor mudança arquitetural capaz de corrigir o problema.

Ordem recomendada:

### Etapa 1 — mapear o código atual

Sem modificar comportamento.

Identificar:

- logout;
- lock;
- heartbeat;
- política;
- revisões;
- comandos;
- status do dispositivo.

### Etapa 2 — substituir logout por bloqueio de sessão

Remover o logout do fluxo normal de enforcement.

Reutilizar o máximo possível da lógica já existente.

### Etapa 3 — separar comando remoto de política

Se atualmente estiverem misturados, separar apenas os pontos necessários.

Não duplicar toda a estrutura de política.

### Etapa 4 — corrigir confirmação de estado

Servidor deve distinguir:

```text
ordem enviada
estado confirmado
```

### Etapa 5 — implementar semântica offline

Garantir:

```text
offline + BLOCK remoto antigo → ignorar
offline + PAUSE remoto antigo → ignorar
offline + política local      → aplicar
```

### Etapa 6 — remover código obsoleto

Depois que o novo fluxo estiver funcionando:

- remover funções de logout que não forem mais usadas;
- remover estados antigos;
- remover caminhos redundantes;
- remover condições criadas apenas para contornar o comportamento anterior.

Essa etapa é obrigatória.

---

# 21. Testes mínimos obrigatórios

## Teste A — documento não salvo

1. Abrir editor.
2. Criar alteração não salva.
3. Encerrar cota.
4. Verificar que sessão é bloqueada.
5. Verificar que aplicativo continua aberto.
6. Desbloquear.
7. Confirmar que documento continua intacto.

---

## Teste B — BLOCK remoto

1. Máquina liberada.
2. Servidor envia BLOCK.
3. Agente bloqueia sessão.
4. Servidor só mostra "Bloqueado" após confirmação real.

---

## Teste C — desbloqueio manual durante BLOCK

1. BLOCK remoto ativo.
2. Usuário desbloqueia com senha.
3. Agente consulta servidor.
4. BLOCK continua vigente.
5. Sessão é bloqueada novamente.

---

## Teste D — perda do servidor durante BLOCK

1. BLOCK remoto ativo.
2. Sessão bloqueada.
3. Derrubar comunicação com servidor.
4. Agente deixa de considerar BLOCK remoto como autoridade.
5. Agente avalia somente política local.
6. Se política permitir uso, sessão não deve continuar bloqueada por causa do comando remoto antigo.

---

## Teste E — PAUSE remoto

1. Cota em andamento.
2. Ativar PAUSE.
3. Confirmar congelamento da contagem.
4. Manter comunicação.
5. Remover PAUSE.
6. Confirmar retomada da contagem.

---

## Teste F — perda do servidor durante PAUSE

1. PAUSE ativo.
2. Derrubar servidor.
3. Agente deixa a pausa.
4. Volta a contabilizar conforme política local.
5. Rotinas voltam a ser aplicadas.

---

## Teste G — política offline

1. Sincronizar política.
2. Derrubar servidor.
3. Atingir fim da cota.
4. Confirmar bloqueio local.
5. Confirmar que nenhuma dependência de comando remoto é necessária.

---

# 22. Critério de conclusão

A melhoria só pode ser considerada concluída quando:

- logout deixar de ser o mecanismo normal de bloqueio;
- documentos não salvos permanecerem preservados;
- BLOCK e PAUSE estiverem separados da política;
- comandos remotos não persistirem como autoridade offline;
- política continuar funcionando offline;
- servidor não confundir ordem enviada com estado aplicado;
- usuário poder desbloquear a sessão sem quebrar o enforcement;
- o agente conseguir bloquear novamente quando necessário;
- código antigo tornado desnecessário tiver sido removido;
- a solução final tiver menos caminhos especiais do que a solução anterior, sempre que possível.

---

# 23. Regra final para Codex

Ao trabalhar neste problema:

> **Não remende o fluxo atual. Entenda a origem do comportamento e substitua a abstração incorreta.**

Antes de adicionar linhas:

> **Pergunte se alguma linha existente deveria ser alterada ou removida.**

Antes de criar novo estado:

> **Verifique se ele já é representado por política, comando remoto ou estado real.**

Antes de criar novo mecanismo:

> **Verifique se o sistema operacional ou a arquitetura atual já oferecem o mecanismo necessário.**

E sempre:

> **Use o mínimo de código, arquivos e tokens necessários para produzir uma correção completa e verificável.**
