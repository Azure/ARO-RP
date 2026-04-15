package steps_test

import (
	"context"
	"testing"
	"time"

	"github.com/Azure/ARO-RP/pkg/util/steps"
	"github.com/sirupsen/logrus"
)


func NewResizeVmStep(vmName string, size string) func(context.Context) error {
	return func(ctx context.Context) error {
		myc := ctx.(resizeContext)
		myc.Log().Warnf("Start Resize vm %s to %s", vmName, size)
		time.Sleep(1 * time.Second)
		myc.Log().Warnf("Stop Resize vm %s to %s", vmName, size)
		return nil
	}
}

func DoJustOneThing(ctx context.Context) error {
	myc := ctx.(resizeContext)
	myc.Log().Warnf("Start YEAH FROM STEP!")
	time.Sleep(1 * time.Second)
	myc.Log().Warnf("Stop YEAH FROM STEP!")
	return nil
}

func AnotherThingy(ctx context.Context) error {
	myc := ctx.(resizeContext)
	myc.Log().Warnf("Start do another")
	time.Sleep(2 * time.Second)
	myc.Log().Warnf("Stop do another")
	return nil
}

func ThirdThingy(ctx context.Context) error {
	myc := ctx.(resizeContext)
	myc.Log().Warnf("Start third")
	time.Sleep(3 * time.Second)
	myc.Log().Warnf("Stop third")
	return nil
}

func ShouldI(ctx context.Context) (bool, error) {
	myc := ctx.(resizeContext)
	myc.Log().Warnf("Start Checking thing")
	time.Sleep(9 * time.Second)
	myc.Log().Warnf("Stop Checking thing")
	return false, nil
}

func ValidateStuff(ctx context.Context) error {
	myc := ctx.(resizeContext)
	myc.Log().Warnf("Start ValidateStuff")
	time.Sleep(2 * time.Second)
	myc.Log().Warnf("Stop Checking thing")
	return nil
	// return fmt.Errorf("precheck failed, cluster api not available")
}

type resizeContext interface {
	context.Context
	Log() *logrus.Entry
}

func Test_StepsThing(t *testing.T) {
	t.Logf("Starting step test")
	testLogger := logrus.New()
	testLogger.SetOutput(t.Output())
	ct := context.Background()
	tle := testLogger.WithContext(ct)
	runContext := NewResizeContext(ct, tle)
	timings, err := steps.Run(runContext, tle, 2*time.Second, []steps.Step{
		steps.Action(DoJustOneThing),
		steps.Action(AnotherThingy),
		steps.Concurrent([]steps.Step{
			steps.Action(DoJustOneThing),
			steps.Action(AnotherThingy),
			steps.Action(ValidateStuff),
		}),
		steps.Action(NewResizeVmStep("master-0", "Standard_d16s")),
	}, time.Now)

	if err != nil {
		t.Errorf("Error in steprun: %v", err)
		t.FailNow()
	}
	t.Errorf("%+v", timings)
}

func NewResizeContext(ctx context.Context, log *logrus.Entry) resizeContext {
	return &rc{
		originalCtx: ctx,
		log:         log,
	}
}

type rc struct {
	originalCtx context.Context
	log         *logrus.Entry
}

// Log implements [resizeContext].
func (r *rc) Log() *logrus.Entry {
	return r.log
}

// Deadline implements [context.Context].
func (r *rc) Deadline() (deadline time.Time, ok bool) {
	return r.originalCtx.Deadline()
}

// Done implements [context.Context].
func (r *rc) Done() <-chan struct{} {
	return r.originalCtx.Done()
}

// Err implements [context.Context].
func (r *rc) Err() error {
	return r.originalCtx.Err()
}

// Value implements [context.Context].
func (r *rc) Value(key any) any {
	return r.originalCtx.Value(key)
}
