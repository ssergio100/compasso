# Avisos de componentes de terceiros

O Compasso usa componentes de terceiros que permanecem sob suas respectivas
licenças. Esta lista cobre as dependências diretas incorporadas aos binários e
ao frontend distribuído pela versão atual. Os textos correspondentes estão em
[`licenses/third-party/`](licenses/third-party/).

| Componente | Versão resolvida | Licença |
| --- | --- | --- |
| [godbus/dbus](https://github.com/godbus/dbus) | 5.1.0 | BSD 3-Clause |
| [mattn/go-sqlite3](https://github.com/mattn/go-sqlite3) | 1.14.32 | MIT |
| [golang.org/x/crypto](https://pkg.go.dev/golang.org/x/crypto) | 0.23.0 | BSD 3-Clause |
| [golang.org/x/sys](https://pkg.go.dev/golang.org/x/sys) | 0.20.0 | BSD 3-Clause |
| [React](https://github.com/facebook/react) | 19.2.8 | MIT |
| [React DOM](https://github.com/facebook/react) | 19.2.8 | MIT |
| [Lucide React](https://github.com/lucide-icons/lucide) | 0.468.0 | ISC |

Ferramentas usadas somente durante o desenvolvimento e a construção continuam
identificadas pelos arquivos `go.mod`, `go.sum`, `admin-ui/package.json` e
`admin-ui/package-lock.json`. Dependências fornecidas pelo sistema operacional
não são redistribuídas neste repositório.
