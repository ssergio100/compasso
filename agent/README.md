# Agent

Código do `tempo-agent`. O pacote `policy` é o motor de decisão puro. A
persistência, o daemon e as integrações Linux serão adicionados nas próximas
fases.

As migrações em `storage/migrations` definem a evolução do banco local. O pacote
`storage` usa SQLite em modo WAL com transações para políticas, checkpoints de
consumo, bônus offline e eventos pendentes.
