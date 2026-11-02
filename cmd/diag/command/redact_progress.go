// Copyright 2026 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package command

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/pingcap/diag/pkg/redact"
	"github.com/pingcap/tiup/pkg/tui/progress"
)

// The existing TiUP renderer owns and joins its refresh goroutine. Only use it
// on a terminal; redirected output gets one start/end line per pass instead.
type redactionProgress struct {
	bar      *progress.SingleBar
	writer   io.Writer
	terminal bool
	phase    string
	started  time.Time
}

func newRedactionProgress() *redactionProgress {
	return &redactionProgress{writer: os.Stdout, terminal: isatty.IsTerminal(os.Stdout.Fd())}
}

func (p *redactionProgress) update(state redact.Progress) {
	if state.Phase != p.phase {
		p.phase = state.Phase
		p.started = time.Now()
		if p.terminal {
			p.bar = progress.NewSingleBar("IP redaction: " + p.phase)
			p.bar.StartRenderLoop()
		} else {
			_, _ = fmt.Fprintf(p.writer, "IP redaction: %s...\n", p.phase)
		}
	}
	detail := redactionProgressDetail(state, time.Since(p.started))
	if p.bar != nil {
		mode := progress.ModeSpinner
		if state.Done {
			mode = progress.ModeDone
		}
		p.bar.UpdateDisplay(&progress.DisplayProps{
			Prefix: "IP redaction: " + p.phase,
			Suffix: detail,
			Detail: detail,
			Mode:   mode,
		})
		if state.Done {
			p.bar.StopRenderLoop()
			p.bar = nil
		}
	} else if state.Done {
		_, _ = fmt.Fprintf(p.writer, "IP redaction: %s done: %s\n", p.phase, detail)
	}
}

func (p *redactionProgress) close(err error) {
	if p.bar != nil {
		p.bar.UpdateDisplay(&progress.DisplayProps{Prefix: "IP redaction: " + p.phase, Mode: progress.ModeError})
		p.bar.StopRenderLoop()
		p.bar = nil
	}
	if err != nil {
		_, _ = fmt.Fprintln(p.writer, "IP redaction failed; see the error below.")
	}
}

func redactionProgressDetail(state redact.Progress, elapsed time.Duration) string {
	percent := 0
	if state.TotalFiles > 0 {
		percent = int(100 * float64(state.Files) / float64(state.TotalFiles))
	}
	// Finishing the last file does not mean the whole phase has completed.
	if percent > 99 {
		percent = 99
	}
	if state.Done {
		percent = 100
	}
	filled := percent / 10
	return fmt.Sprintf("[%s%s] %3d%% | %d/%d files | %s",
		strings.Repeat("=", filled), strings.Repeat(" ", 10-filled), percent,
		state.Files, state.TotalFiles, elapsed.Round(time.Second))
}
