package agent

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/bitcode-engine/engine/internal/compiler/parser"
)

type CronJob struct {
	Schedule string
	Script   string
	Action   string
}

type CronScheduler struct {
	jobs         []CronJob
	scriptRunner ScriptRunner
	stopCh       chan struct{}
	mu           sync.Mutex
}

func NewCronScheduler(scriptRunner ScriptRunner) *CronScheduler {
	return &CronScheduler{
		scriptRunner: scriptRunner,
		stopCh:       make(chan struct{}),
	}
}

func (s *CronScheduler) RegisterAgent(agentDef *parser.AgentDefinition) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, cron := range agentDef.Cron {
		s.jobs = append(s.jobs, CronJob{
			Schedule: cron.Schedule,
			Script:   cron.Script,
			Action:   cron.Action,
		})
	}
}

func (s *CronScheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.runDueJobs(ctx)
		}
	}
}

func (s *CronScheduler) Stop() {
	close(s.stopCh)
}

func (s *CronScheduler) runDueJobs(ctx context.Context) {
	s.mu.Lock()
	jobs := make([]CronJob, len(s.jobs))
	copy(jobs, s.jobs)
	s.mu.Unlock()

	for _, job := range jobs {
		if _, err := s.scriptRunner.Run(ctx, job.Script, nil); err != nil {
			log.Printf("[cron] job %s failed: %v", job.Action, err)
		}
	}
}
