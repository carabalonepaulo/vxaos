package game

import (
	"context"
	"log/slog"
	"sync"

	"github.com/kelaia/vxaos/server/internal/store"
)

// persistJob is an immutable snapshot handed to the persistence worker. The
// world/map goroutine must never share mutable character state with the worker.
type persistJob struct {
	character *store.Character
	state     *store.CharacterState
}

type persistWorker struct {
	store *store.Store
	log   *slog.Logger
	jobs  chan persistJob
	done  chan struct{}
	once  sync.Once
}

func newPersistWorker(st *store.Store, log *slog.Logger) *persistWorker {
	p := &persistWorker{
		store: st,
		log:   log,
		jobs:  make(chan persistJob, 1024),
		done:  make(chan struct{}),
	}
	go p.loop()
	return p
}

func (p *persistWorker) loop() {
	defer close(p.done)
	for job := range p.jobs {
		if job.character == nil || job.state == nil {
			continue
		}
		if err := p.store.SaveCharacter(context.Background(), job.character, job.state); err != nil {
			p.log.Error("persist character", "character", job.character.ID, "err", err)
		}
	}
}

// queue copies the mutable character/state and hands the copy to the worker.
func (p *persistWorker) queue(character *store.Character, state *store.CharacterState) {
	if character == nil || state == nil {
		return
	}
	job := persistJob{
		character: cloneCharacter(character),
		state:     cloneCharacterState(state),
	}
	select {
	case p.jobs <- job:
	default:
		// A full queue is a persistence emergency; do not silently lose data.
		p.log.Error("persist queue full", "character", character.ID)
	}
}

// closeAndFlush stops accepting work and waits for queued saves to finish.
func (p *persistWorker) closeAndFlush(ctx context.Context) error {
	p.once.Do(func() { close(p.jobs) })
	select {
	case <-p.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func cloneCharacter(c *store.Character) *store.Character {
	if c == nil {
		return nil
	}
	out := *c
	return &out
}

func cloneCharacterState(st *store.CharacterState) *store.CharacterState {
	if st == nil {
		return nil
	}
	out := *st
	out.Equips = append([]int(nil), st.Equips...)
	out.Inventory = append([]store.ItemStack(nil), st.Inventory...)
	out.Skills = append([]int(nil), st.Skills...)
	out.Quests = append([]store.QuestState(nil), st.Quests...)
	out.Hotbar = append([]store.HotbarSlot(nil), st.Hotbar...)
	out.Switches = append([]bool(nil), st.Switches...)
	out.Variables = append([]int32(nil), st.Variables...)
	out.SelfSwitches = append([]store.SelfSwitchState(nil), st.SelfSwitches...)
	return &out
}
