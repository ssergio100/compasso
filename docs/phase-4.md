# Fase 4 — requisito de login substituído

> O gate PAM descrito abaixo pertence ao protótipo anterior e permanece
> desativado. O requisito atual permite a autenticação do usuário bloqueado;
> depois que a sessão gráfica estiver estabelecida, o agente deve executar um
> logout seguro e retornar ao greeter sem tela preta. Este procedimento não
> deve ser usado no pacote atual.

## Escopo implementado

- `tempo-pam-check` consulta a última política e o consumo persistidos no SQLite;
- somente a conta definida por `controlled_user` é submetida à política;
- cota esgotada, rotina ativa e bloqueio manual recusam acesso;
- vigilância pausada permite acesso;
- a consulta funciona sem rede e mesmo se o processo do agente estiver parado;
- erro interno, configuração ausente ou banco ilegível seguem `fail-open`, evitando
  inutilizar permanentemente a tela de login;
- `tempo-pam-setup` instala o gate somente em `gdm-password`, depois de salvar
  uma cópia integral do arquivo original;
- a desinstalação restaura exatamente a cópia original.

O gate usa a etapa `account` do PAM. Uma regra `pam_succeed_if` pula o helper
para qualquer conta diferente da controlada. A preparação técnica ainda é feita
pelos executáveis de instalação; o teste humano acontece exclusivamente pela
tela gráfica de login do Zorin.

## Registro histórico da preparação técnica

Os comandos abaixo registram como o protótipo era ensaiado e **não devem mais
ser executados**. Antes do teste real, confirmava-se que o display manager usava
`/etc/pam.d/gdm-password`. Depois de compilar e instalar o agente da fase 3:

```bash
make build-agent
sudo install -m 0755 bin/tempo-pam-check /usr/libexec/tempo-pam-check
sudo install -m 0755 bin/tempo-pam-setup /usr/sbin/tempo-pam-setup
sudo tempo-pam-setup -action install
```

O instalador recusa continuar se o helper não estiver instalado e executável.
Não feche a sessão usada na preparação antes de conferir que o backup existe em
`/etc/pam.d/gdm-password.compasso.bak`.

Para remover o gate e restaurar o arquivo anterior:

```bash
sudo tempo-pam-setup -action uninstall
```

## Teste humano substituído

O roteiro abaixo também é apenas histórico e não é critério de aceite atual:

1. Com política liberada, entrar normalmente na conta controlada.
2. Aplicar uma política bloqueada e tentar entrar novamente pela tela do Zorin.
3. Confirmar que o login é recusado e que a tela continua operacional.
4. Pausar a vigilância e confirmar que o login volta a funcionar.
5. Parar temporariamente o agente e confirmar que a última política persistida
   continua sendo consultada.
6. Desinstalar o gate e confirmar que o login original foi restaurado.

Os testes acima só serão marcados como integração real depois de executados no
Zorin. Os checkboxes automatizados na especificação identificam explicitamente
o uso de PAM e arquivos simulados.
