# kubectl-config-cleanup

In my daily work I use many Kubernetes clusters, many of which are short-lived. I usually import them into my main kubeconfig using [corneliusweig/konfig](https://github.com/corneliusweig/konfig) kubectl plugin, which leaves me with way too many stale contexts.

This tool is a solution. It uses [github.com/ktr0731/go-fuzzyfinder](https://github.com/ktr0731/go-fuzzyfinder) library to select kubeconfig contexts to delete, along with all of the users and clusters that the config references. This tools respects `--kubeconfig` flag
and `KUBECONFIG` environment variable. By default it also removes all of the stale users and clusters that are not referenced by any context.

## Installation

```bash
go install aerf.io/kubectl-config-cleanup/cmd/kubectl-config_cleanup@main
```

## Usage

You may use it either as a standalone binary, so as `kubectl-config_cleanup`, or as a kubectl plugin: `kubectl config-cleanup`.
