package worker

import "context"

func (w *Worker) runDailyBriefJob(ctx context.Context) error {
	return w.ensureDailyBriefs(ctx)
}
