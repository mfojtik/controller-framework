package main

import (
	"context"
	"github.com/mfojtik/controller-framework/examples/controllers/embedded"
	"github.com/mfojtik/controller-framework/examples/controllers/errorhandling"
	"github.com/mfojtik/controller-framework/examples/controllers/informer"
	"github.com/mfojtik/controller-framework/examples/controllers/simple"
	"github.com/mfojtik/controller-framework/pkg/events"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	recorder := events.NewInMemoryRecorder("examples")

	fakeInformer := informer.NewFakeInformer(5*time.Second, 0)

	simpleController := simple.New(recorder)
	errorHandlingController := errorhandling.New(recorder)
	embeddedController := embedded.New(recorder)
	informerController := informer.New(fakeInformer, recorder)

	go simpleController.Run(ctx, 1)
	go errorHandlingController.Run(ctx, 1)
	go embeddedController.Run(ctx, 1)
	go informerController.Run(ctx, 1)

	<-ctx.Done() // wait for the context to finish
}
