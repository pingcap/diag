// Copyright 2026 PingCAP, Inc.
// Licensed under the Apache License, Version 2.0.

package redact

import (
	"errors"
	"sync"
)

// sharedState enforces per-pass limits across all workers.
type sharedState struct {
	mu        sync.Mutex
	remaining int64
	entries   int
}

// SetWorkers sets the maximum number of files processed at once.
func (r *Redactor) SetWorkers(n int) error {
	if n < 1 || n > 256 {
		return errors.New("redaction workers must be between 1 and 256")
	}
	r.workers = n
	return nil
}

func (r *Redactor) consume(n int64) bool {
	if r.shared == nil {
		r.remaining -= n
		return r.remaining >= 0
	}
	r.shared.mu.Lock()
	defer r.shared.mu.Unlock()
	r.shared.remaining -= n
	return r.shared.remaining >= 0
}

func (r *Redactor) discoveryCount() int {
	if r.shared != nil {
		r.shared.mu.Lock()
		defer r.shared.mu.Unlock()
	}
	return len(r.discovered)
}

type filePool struct {
	jobs   chan func(*Redactor) error
	wg     sync.WaitGroup
	mu     sync.Mutex
	err    error
	report Report
}

func (r *Redactor) newPool() *filePool {
	if r.shared == nil {
		r.shared = &sharedState{remaining: r.remaining, entries: r.entries}
	}
	n := r.workers
	if n < 1 {
		n = 1
	}
	p := &filePool{jobs: make(chan func(*Redactor) error)}
	for i := 0; i < n; i++ {
		worker := *r
		worker.report = Report{}
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for job := range p.jobs {
				p.mu.Lock()
				failed := p.err != nil
				p.mu.Unlock()
				if failed {
					continue
				}
				if err := job(&worker); err != nil {
					p.mu.Lock()
					if p.err == nil {
						p.err = err
					}
					p.mu.Unlock()
				}
			}
			p.mu.Lock()
			p.report.Files += worker.report.Files
			p.report.Replacements += worker.report.Replacements
			p.report.LogIPv4QuestionMarks += worker.report.LogIPv4QuestionMarks
			p.mu.Unlock()
		}()
	}
	return p
}

func (p *filePool) submit(job func(*Redactor) error) error {
	p.mu.Lock()
	err := p.err
	p.mu.Unlock()
	if err != nil {
		return err
	}
	p.jobs <- job
	return nil
}

func (p *filePool) finish(r *Redactor) error {
	close(p.jobs)
	p.wg.Wait()
	r.report.Files += p.report.Files
	r.report.Replacements += p.report.Replacements
	r.report.LogIPv4QuestionMarks += p.report.LogIPv4QuestionMarks
	return p.err
}
