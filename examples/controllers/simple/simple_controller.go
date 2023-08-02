package simple

import (
	"context"
	"github.com/mfojtik/controller-framework/pkg/events"
	"github.com/mfojtik/controller-framework/pkg/factory"
	"github.com/mfojtik/controller-framework/pkg/framework"
	"time"
)

func New(recorder events.Recorder) framework.Controller {
	return factory.New().WithSync(sync).ResyncEvery(1*time.Second).ToController("simple", recorder)
}

func sync(ctx context.Context, context framework.Context) error {
	defer context.Recorder().Warningf("SimpleReconciled", "reconciling done")
	return nil
}
