### Local Mode


The plugins must be placed in `./plugins-local` directory,
which should be in the working directory of the process running the Traefik binary.
The source code of the plugin should be organized as follows:

```
 └── plugins-local
    └── src
        └── github.com
            └── kav789
                └── traefik-ratelimit
                    ├── main.go
                    ├── go.mod
                    └── ...
```
