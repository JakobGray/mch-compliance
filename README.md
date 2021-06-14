# mch-compliance

This repo tests whether components installed by the multiclusterhub comply with standards defined in
the [chart onboarding documentation](https://github.com/open-cluster-management/multiclusterhub-repo/blob/main/docs/Onboarding.md). 

## Running

Run `go install` to build a CLI named `mch-compliance`. Alternatively run `go run main.go`. The CLI contains a single `audit` commmand to be run against an installed hub cluster.

```
Usage:
  mch-compliance audit [flags]

Examples:
# Define a list of checks to exclude from report
mch-compliance audit -c exemptions.yaml

# Generate report in a parseable json format
mch-compliance audit -o json

Flags:
  -c, --config string      Configuration file with exemptions
  -f, --file string        Filepath to save results
  -h, --help               help for audit
  -n, --namespace string   Namespace where the hub is installed (default "open-cluster-management")
  -o, --output string      Results format to output (default "table")
```

The `examples` folder contains an example config for excluding deployments installed via bundle, and results generated in text and yaml format by calling

``` bash
mch-compliance audit -c examples/exemptions.yaml -f examples/output.yaml -o yaml
mch-compliance audit -c examples/exemptions.yaml -f examples/output.txt -o text
```