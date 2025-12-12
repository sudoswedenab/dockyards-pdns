// Copyright 2025 Sudo Sweden AB
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	"github.com/go-logr/logr"
	"github.com/spf13/pflag"
	dyconfig "github.com/sudoswedenab/dockyards-backend/api/config"
	controllers "github.com/sudoswedenab/dockyards-pdns/controllers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const managementDomainKey = "managementDomain"

// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;patch;watch

func main() {
	var dockyardsNamespace string
	var configMap string
	pflag.StringVar(&configMap, "config-map", "dockyards-system", "ConfigMap name")
	pflag.StringVar(&dockyardsNamespace, "dockyards-namespace", "dockyards-system", "dockyards namespace")
	pflag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slogr := logr.FromSlogHandler(logger.Handler())

	ctrl.SetLogger(slogr)

	cfg, err := config.GetConfig()
	if err != nil {
		logger.Error("error getting config", "err", err)

		os.Exit(1)
	}

	m, err := manager.New(cfg, manager.Options{})
	if err != nil {
		logger.Error("error creating manager", "err", err)

		os.Exit(1)
	}

	scheme := runtime.NewScheme()

	_ = corev1.AddToScheme(scheme)

	controllerClient, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		logger.Error("error creating new controller client", "err", err)

		os.Exit(1)
	}

	dockyardsConfig, err := dyconfig.GetConfig(ctx, controllerClient, configMap, dockyardsNamespace)
	if err != nil {
		logger.Error("error getting dockyards config", "err", err)

		os.Exit(1)
	}

	md := dockyardsConfig.GetConfigKey(managementDomainKey, "")
	if md == "" {
		logger.Error("no " + managementDomainKey + " specified in dockyards config")

		os.Exit(1)
	}

	err = (&controllers.DockyardsClusterReconciler{
		Client:                m.GetClient(),
		DockyardsConfigReader: dockyardsConfig,
	}).SetupWithManager(m)
	if err != nil {
		logger.Error("error creating new dockyards cluster reconciler", "err", err)

		os.Exit(1)
	}

	err = (&controllers.ZoneReconciler{
		Client:                m.GetClient(),
		DockyardsConfigReader: dockyardsConfig,
	}).SetupWithManager(m)
	if err != nil {
		logger.Error("error creating new zone reconciler", "err", err)

		os.Exit(1)
	}

	err = m.Start(ctx)
	if err != nil {
		logger.Error("error running manager", "err", err)

		os.Exit(1)
	}
}
