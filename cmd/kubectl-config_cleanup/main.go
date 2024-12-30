package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"slices"

	"golang.org/x/exp/maps"

	"github.com/ktr0731/go-fuzzyfinder"
	"github.com/spf13/pflag"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/yaml"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	pathOptions := clientcmd.NewDefaultPathOptions()
	removeStaleUsers := true
	removeStaleClusters := true
	pflag.StringVar(&pathOptions.LoadingRules.ExplicitPath, pathOptions.ExplicitFileFlag, pathOptions.LoadingRules.ExplicitPath, "Use a particular kubeconfig file")
	pflag.BoolVarP(&removeStaleUsers, "remove-stale-users", "u", true, "Remove stale users, not referenced by any context")
	pflag.BoolVarP(&removeStaleClusters, "remove-stale-clusters", "c", true, "Remove stale clusters, not referenced by any context")
	help := pflag.BoolP("help", "h", false, "Print help message")
	pflag.Parse()

	if *help {
		pflag.Usage()
		os.Exit(0)
	}
	cfg, err := pathOptions.GetStartingConfig()
	if err != nil {
		return err
	}

	namedContexts := k8sConfigContexts(cfg.Contexts)
	idxs, err := fuzzyfinder.FindMulti(
		namedContexts,
		func(i int) string {
			return namedContexts[i].Name
		},
		fuzzyfinder.WithPreviewWindow(func(i, _, _ int) string {
			if i == -1 {
				return ""
			}

			objBytes, err := json.Marshal(namedContexts[i].Context)
			if err != nil {
				panic(err)
			}
			yamlBytes, err := yaml.JSONToYAML(objBytes)
			if err != nil {
				panic(err)
			}

			return string(yamlBytes)
		}),
		fuzzyfinder.WithHeader("Use tab to select contexts to remove. Users and clusters that they reference will be also removed"),
		fuzzyfinder.WithCursorPosition(fuzzyfinder.CursorPositionTop),
	)
	if err != nil {
		if errors.Is(err, fuzzyfinder.ErrAbort) {
			return nil
		}
		return fmt.Errorf("failed to fuzzy-search through kubeconfig: %s", err)
	}

	clustersToDelete := []string{}
	usersToDelete := []string{}
	for _, i := range idxs {
		ctx := namedContexts[i]
		delete(cfg.Contexts, ctx.Name)
		usersToDelete = append(usersToDelete, ctx.Context.AuthInfo)
		clustersToDelete = append(clustersToDelete, ctx.Context.Cluster)
	}

	for _, cluster := range clustersToDelete {
		delete(cfg.Clusters, cluster)
	}
	maps.DeleteFunc(cfg.Clusters, func(key string, _ *api.Cluster) bool {
		return slices.Contains(clustersToDelete, key)
	})
	maps.DeleteFunc(cfg.AuthInfos, func(key string, _ *api.AuthInfo) bool {
		return slices.Contains(usersToDelete, key)
	})
	if removeStaleClusters {
		staleClusters := []string{}
		allClusters := maps.Keys(cfg.Clusters)
		referencedClusters := []string{}
		for _, kubeCtx := range cfg.Contexts {
			referencedClusters = append(referencedClusters, kubeCtx.Cluster)
		}
		for _, cluster := range allClusters {
			if !slices.Contains(referencedClusters, cluster) {
				staleClusters = append(staleClusters, cluster)
			}
		}

		for _, stale := range staleClusters {
			delete(cfg.Clusters, stale)
		}
	}

	if removeStaleUsers {
		staleUsers := []string{}
		allUsers := maps.Keys(cfg.AuthInfos)
		referencedUsers := []string{}
		for _, kubeCtx := range cfg.Contexts {
			referencedUsers = append(referencedUsers, kubeCtx.AuthInfo)
		}
		for _, user := range allUsers {
			if !slices.Contains(referencedUsers, user) {
				staleUsers = append(staleUsers, user)
			}
		}

		for _, stale := range staleUsers {
			delete(cfg.AuthInfos, stale)
		}
	}

	return clientcmd.ModifyConfig(pathOptions, *cfg, true)
}

type NamedAPIContext struct {
	Name    string
	Context *api.Context
}

func k8sConfigContexts(ctxs map[string]*api.Context) []NamedAPIContext {
	out := make([]NamedAPIContext, 0, len(ctxs))
	for key, ctx := range ctxs {
		out = append(out, NamedAPIContext{
			Name:    key,
			Context: ctx,
		})
	}
	return out
}
