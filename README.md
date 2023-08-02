# Simple and Flexible Kubernetes Controller Development Kit

[![CodeQL](https://github.com/mfojtik/controller-framework/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/mfojtik/controller-framework/actions/workflows/github-code-scanning/codeql)
[![Dependency Review](https://github.com/mfojtik/controller-framework/actions/workflows/dependency-review.yml/badge.svg)](https://github.com/mfojtik/controller-framework/actions/workflows/dependency-review.yml)
[![Go](https://github.com/mfojtik/controller-framework/actions/workflows/go.yml/badge.svg)](https://github.com/mfojtik/controller-framework/actions/workflows/go.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/mfojtik/controller-framework.svg)](https://pkg.go.dev/github.com/mfojtik/controller-framework)

The **controller-framework** is a versatile Golang module designed to facilitate the creation of robust and feature-rich [Kubernetes controllers](https://kubernetes.io/docs/concepts/architecture/controller). This framework offers a straightforward approach to building controllers with essential functionalities, including graceful shutdown handling, informer synchronization, and work queue management. Contrasted with the [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) library, the controller-framework prioritizes simplicity and flexibility, making it an ideal choice for crafting controllers that closely resemble the traditional ones used within the Kubernetes project.

## Features

- **Graceful Shutdown**: The Controller Framework ensures a smooth and graceful shutdown of your controllers. It allows ongoing workers to finalize and resources to be managed appropriately before the controller terminates.

- **Informer Synchronization**: Easily synchronize informers with the controller-framework, enabling seamless tracking and reaction to changes in Kubernetes resources. The module abstracts the complexity of informer cache synchronization.

- **Workqueue Management**: The framework streamlines work queue management, allowing tasks to be efficiently queued and processed. It handles retries and back-off strategies for any failed tasks, promoting reliability.

- **Simplicity and Flexibility**: Prioritizing simplicity, the controller-framework minimizes boilerplate code, empowering developers to focus on implementing the core logic of their controllers.

- **Familiar Kubernetes Controller Paradigm**: Controllers created using this framework mirror the established structure and design principles of traditional Kubernetes controllers. Developers familiar with Kubernetes controller development will find it intuitive to work with this framework.

## Installation

Start to integrate the controller framework into your Golang project by using `factory` Go modules:

```bash
import "github.com/mfojtik/controller-framework/pkg/factory"
```

The factory provide a builder that produce all Kubernetes controller boilerplate. To learn how flexible it is, read the  [documentation](https://pkg.go.dev/github.com/mfojtik/controller-framework@master/pkg/factory)

```go
func New() framework.controller {
    return factory.New().ToController("new-controller", eventRecorder)	
}
```

You can access the workqueue inside the `sync()` function via the [controller context](https://pkg.go.dev/github.com/mfojtik/controller-framework@master/pkg/context).

```go
func (c *controller) sync(ctx context.Context, syncCtx factory.SyncContext) error {
	syncCtx.Recorder().Eventf(...)
	syncCtx.Queue().Add("key")
}
```

And finally, for Kubernetes Event management, you can use included [Event Recorder](https://pkg.go.dev/github.com/mfojtik/controller-framework@master/pkg/events) which offers various
backends for storing events (filesystem, [Kubernetes Events](https://pkg.go.dev/k8s.io/client-go@v0.27.4/kubernetes/typed/events/v1#NewForConfig), in-memory events, etc.)
```go
...

recorder := events.NewRecorder(client.CoreV1().Events("test-namespace"), "test-operator", controllerRef)

...
```

## Getting Started

Utilizing the controller framework is straightforward. Here's a simple example showcasing how to create a controller:

```go
// simple_controller.go:
package simple

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/mfojtik/controller-framework/pkg/events"
	"github.com/mfojtik/controller-framework/pkg/factory"
	"github.com/mfojtik/controller-framework/pkg/framework"
)

type controller struct {}

func New(recorder events.Recorder) framework.Controller {
	c := &controller{}
	return factory.New().
		WithSync(c.sync).                // reconcile function
		// WithInformers(secretInformer) // react to secretInformer changes
		ResyncEvery(10*time.Second).     // queue sync() every 10s regardless of informers
		ToController("simple", recorder)
}

func (c *controller) sync(ctx context.Context, controllerContext framework.Context) error {
	// do stuff
	_, err := os.ReadFile("/etc/cert.pem")
	if errors.Is(err, os.ErrNotExist) {
		// controller will requeue and retry
		return errors.New("file not found")
	}
	return nil
}

// main.go:

func main() {
	// the controllerRef is an Kubernetes ownerReference to an object the events will tied to (a pod, namespace, etc)
	recorder := events.NewRecorder(client.CoreV1().Events("test-namespace"), "test-operator", controllerRef)
	
	controller := simple.New(recorder)
	
	// when this context is done, the controllers will gracefully shutddown
	ctx := context.Background()
	
	// start the controller with one worker
	go controller.Run(ctx, 1)
	...
}
```

Check the [Examples](https://github.com/mfojtik/controller-framework/tree/master/examples) for more controller examples.

## Contribution

Contributions to the controller-framework are welcomed! Feel free to submit issues and pull requests via the [GitHub repository](https://github.com/mfojtik/controller-framework).

## License

The Controller Framework is distributed under the Apache v2 License. Refer to the [LICENSE](LICENSE) file for more details.
