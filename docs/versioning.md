# Versionamento

O projeto usa Versionamento Semântico (`MAJOR.MINOR.PATCH`). Durante o
desenvolvimento da baseline, a versão é `0.1.0-dev`, registrada no arquivo
`VERSION`.

- `MAJOR`: mudança incompatível no contrato ou no formato persistido.
- `MINOR`: funcionalidade compatível adicionada.
- `PATCH`: correção compatível.

Releases usam tags Git anotadas no formato `v0.1.0`. A versão da API (`/api/v1`),
a revisão de política por dispositivo e a versão das migrações são contadores
independentes da versão do binário.
