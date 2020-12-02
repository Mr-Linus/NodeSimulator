/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	scv1 "github.com/NJUPT-ISL/SCV/api/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
	"os"

	simv1 "github.com/NJUPT-ISL/NodeSimulator/pkg/api/v1"
	"github.com/NJUPT-ISL/NodeSimulator/pkg/controllers/node"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = scv1.AddToScheme(scheme)
	_ = simv1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	ctrl.SetLogger(zap.New(func(o *zap.Options) {
		o.Development = true
	}))

	mgrConfig := ctrl.GetConfigOrDie()
	mgrConfig.QPS = 1000
	mgrConfig.Burst = 1000
	mgr, err := ctrl.NewManager(mgrConfig, ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		LeaderElection:     enableLeaderElection,
		Port:               9443,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Init ClientSet
	clientSet, err := kubernetes.NewForConfig(mgrConfig)
	if err != nil {
		setupLog.Error(err, "unable to init clientSet")
		os.Exit(1)
	}

	if err = (&node.NodeSimReconciler{
		Client:    mgr.GetClient(),
		ClientSet: clientSet,
		Log:       ctrl.Log.WithName("controllers").WithName("NodeSimulator"),
		Scheme:    mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "NodeSimulator")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	stopChan := make(chan struct{}, 0)
	nodeUpdater, err := node.NewNodeUpdater(mgr.GetClient(),
		workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		stopChan)

	if err == nil {
		go nodeUpdater.Run(5, stopChan)
	} else {
		klog.Errorf("New NodeUpdate Error: %v", err)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
