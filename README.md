# controller-framework: Simple and Flexible Kubernetes Controller Development

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

Integrate the controller framework into your Golang project by using Go modules:

```bash
import "github.com/mfojtik/controller-framework/pkg/factory"
```

## Getting Started

Utilizing the controller framework is straightforward. Here's a basic example showcasing how to create a controller:

```go
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mfojtik/controller-framework"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

```

## Contribution

Contributions to the controller-framework are welcomed! Feel free to submit issues and pull requests via the [GitHub repository](https://github.com/mfojtik/controller-framework).

## License

The Controller Framework is distributed under the Apache v2 License. Refer to the [LICENSE](LICENSE) file for more details.
