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

type bankJob struct {
	accountID int64
	bank      *store.BankState
}

type persistWorker struct {
	store *store.Store
	log   *slog.Logger
	jobs  chan persistJob
	banks chan bankJob
	done  chan struct{}
	once  sync.Once
}

func newPersistWorker(st *store.Store, log *slog.Logger) *persistWorker {
	p := &persistWorker{
		store: st,
		log:   log,
		jobs:  make(chan persistJob, 1024),
		banks: make(chan bankJob, 128),
		done:  make(chan struct{}),
	}
	go p.loop()
	return p
}

func (p *persistWorker) loop() {
	defer close(p.done)
	for p.jobs != nil || p.banks != nil {
		select {
		case job, ok := <-p.jobs:
			if !ok {
				p.jobs = nil
				continue
			}
			if job.character == nil || job.state == nil {
				continue
			}
			if err := p.store.SaveCharacter(context.Background(), job.character, job.state); err != nil {
				p.log.Error("persist character", "character", job.character.ID, "err", err)
			}
		case job, ok := <-p.banks:
			if !ok {
				p.banks = nil
				continue
			}
			if job.bank == nil {
				continue
			}
			if err := p.store.SaveBank(context.Background(), job.accountID, job.bank); err != nil {
				p.log.Error("persist bank", "account", job.accountID, "err", err)
			}
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

func (p *persistWorker) queueBank(accountID int64, bank *store.BankState) {
	if bank == nil {
		return
	}
	job := bankJob{accountID: accountID, bank: cloneBankState(bank)}
	select {
	case p.banks <- job:
	default:
		p.log.Error("bank persist queue full", "account", accountID)
	}
}

// closeAndFlush stops accepting work and waits for queued saves to finish.
func (p *persistWorker) closeAndFlush(ctx context.Context) error {
	p.once.Do(func() {
		close(p.jobs)
		close(p.banks)
	})
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
