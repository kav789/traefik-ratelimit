### Local Mode


The plugins must be placed in `./plugins-local` directory,
which should be in the working directory of the process running the Traefik binary.
The source code of the plugin should be organized as follows:

```
 ├── docker-compose.yml
 └── plugins-local
    └── src
        └── github.com
            └── kav789
                └── traefik-ratelimit
                    ├── main.go
                    ├── go.mod
                    └── ...

```
parameters:

```
  - keeperRateLimitKey=wbpay-ratelimits
  - keeperURL=http://keeper-ext.wbpay.svc.k8s.wbpay-dev:8080
  - keeperAdminPassword=Pa$sw0rd
  - keeperReqTimeout=300s

```
keeper:

```
{
  "limits": [
    {"endpointpat": "/api/v2/methods",         "limit": 1},
    {"endpointpat": "/api/v2/methods",         "limit": 2},
    {"endpointpat": "/api/v2/**/methods",     "headerkey": "aa-bb", "headerval": "AsdfG", "limit": 1},
    {"endpointpat": "/api/v2/*/aa/**/methods", "limit": 1}
  ]
}

```
