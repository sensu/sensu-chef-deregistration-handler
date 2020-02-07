[![Bonsai Asset Badge](https://img.shields.io/badge/Sensu%20Chef%20Deregistration%20Handler-Download%20Me-brightgreen.svg?colorB=89C967&logo=sensu)](https://bonsai.sensu.io/assets/sensu/sensu-chef-deregistration-handler)
![Go Test](https://github.com/sensu/sensu-chef-deregistration-handler/workflows/Go%20Test/badge.svg)

# Sensu Chef Deregistration Handler

- [Overview](#overview)
- [Usage examples](#usage-examples)
- [Configuration](#configuration)
  - [Asset registration](#asset-registration)
  - [Handler definition](#handler-definition)
  - [Check definition](#check-definition)
- [Installation from source and
  contributing](#installation-from-source-and-contributing)

## Overview

The [Sensu Chef Deregistration Handler][0] is a [Sensu Event Handler][3] that
will delete an entity with a failing keepalive check when its corresponding
[Chef][2] node no longer exists.

## Usage examples

Help:

```
Usage:
  sensu-chef-deregistration-handler [flags]
  sensu-chef-deregistration-handler [command]

Available Commands:
  help        Help about any command
  version     Print the version number of this plugin

Flags:
  -k, --client-key-path string   The path to the Chef Client key to use when authenticating/querying the Chef Server API
  -c, --client-name string       The Chef Client name to use when authenticating/querying the Chef Server API
  -e, --endpoint string          The Chef Server API endpoint (URL)
  -f, --flavor string            The Chef Server flavor (enterprise or open_source)
  -h, --help                     help for sensu-chef-deregistration-handler
      --sensu-api-key string     The Sensu API key
      --sensu-api-url string     The Sensu API URL (default "http://localhost:8080")
      --sensu-ca-cert string     The Sensu Go CA Certificate
  -p, --ssl-pem-path string      The Chef SSL pem file use when querying the Chef Server API
  -s, --ssl-verify               If the SSL certificate will be verified when querying the Chef Server API (default true)
```

## Configuration

### Asset registration

Assets are the best way to make use of this handler. If you're not using an asset, please consider doing so! If you're using sensuctl 5.13 or later, you can use the following command to add the asset:

`sensuctl asset add sensu/sensu-chef-deregistration-handler`

If you're using an earlier version of sensuctl, you can download the asset
definition from [this project's Bonsai Asset Index
page](https://bonsai.sensu.io/assets/sensu/sensu-chef-deregistration-handler).

### Handler definition

Create the handler using the following handler definition:

```yml
---
api_version: core/v2
type: Handler
metadata:
  namespace: default
  name: sensu-chef-deregistration-handler
spec:
  type: pipe
  command: sensu-chef-deregistration-handler
  timeout: 10
  env_vars:
  - CHEF_ENDPOINT=https://api.chef.io/organizations/replace-me
  - CHEF_CLIENT_NAME=your-chef-client
  - CHEF_CLIENT_KEY_PATH=/path/to/chef/client.pem
  - SENSU_API_URL=https://sensu-backend-url:8080
  - SENSU_API_KEY=sensu-api-key-here
  filters:
  - is_incident
  runtime_assets:
  - sensu/sensu-chef-deregistration-handler
```

and then add the handler to the keepalive handler set:

``` yml
---
api_version: core/v2
type: Handler
metadata:
  name: keepalive
  namespace: default
spec:
  handlers:
  - sensu-chef-deregistration-handler
  type: set
```


### Check definition

No check definition is needed. This handler will only trigger on keepalive
events after it is added to the keepalive handler set.

## Installing from source and contributing

Download the latest version of the sensu-chef-deregistration-handler from [releases][4],
or create an executable script from this source.

### Compiling

From the local path of the sensu-chef-deregistration-handler repository:
```
go build -o /usr/local/bin/ .
```

To contribute to this plugin, see [CONTRIBUTING](https://github.com/sensu/sensu-go/blob/master/CONTRIBUTING.md)

[0]: https://github.com/sensu/sensu-chef-deregistration-handler
[1]: https://github.com/sensu/sensu-go
[2]: https://chef.io
[3]: https://docs.sensu.io/sensu-go/latest/reference/handlers/#how-do-sensu-handlers-work
[4]: https://github.com/sensu/sensu-chef-deregistration-handler/releases
