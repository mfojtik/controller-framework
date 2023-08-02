package errorhandling

import (
	"context"
	"errors"
	"github.com/mfojtik/controller-framework/pkg/events"
	"github.com/mfojtik/controller-framework/pkg/factory"
	"github.com/mfojtik/controller-framework/pkg/framework"
	"math/rand"
)

type controller struct {
	counter int
}

func (c *controller) sync(ctx context.Context, context framework.Context) error {
	v := rand.Int()
	if isEven(v) {
		return errors.New("even number, will retry until odd number")
	}
	context.Recorder().Warningf("OddNumber", "got odd number %i", v)
	return nil
}

func New(recorder events.Recorder) framework.Controller {
	c := &controller{}
	return factory.New().WithSync(c.sync).ResyncSchedule("@every 5s").ToController("error-handling", recorder)
}

func isEven(number int) bool {
	return number%2 == 0
}
