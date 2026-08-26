//go:build gohook

package input

import (
	"context"
	"errors"
	"fmt"
	"sync"

	hook "github.com/robotn/gohook"
)

// GlobalSource adapts OS-wide keyboard and mouse hooks to Source.
type GlobalSource struct {
	events <-chan hook.Event
	once   sync.Once
}

// NewGlobalSource starts global keyboard and mouse monitoring.
func NewGlobalSource() (Source, error) {
	events := hook.Start()
	if events == nil {
		return nil, errors.New("start global input hook")
	}
	return &GlobalSource{events: events}, nil
}

// Read waits for the next keyboard or mouse event and normalizes it.
func (source *GlobalSource) Read(ctx context.Context) (Event, error) {
	if source == nil || source.events == nil {
		return Event{}, errors.New("global input source is not initialized")
	}
	select {
	case <-ctx.Done():
		return Event{}, ctx.Err()
	case event, ok := <-source.events:
		if !ok {
			return Event{}, errors.New("global input hook stopped")
		}
		return normalizeHookEvent(event)
	}
}

// Close stops the global hook exactly once.
func (source *GlobalSource) Close() error {
	if source == nil {
		return nil
	}
	source.once.Do(hook.End)
	return nil
}

func normalizeHookEvent(event hook.Event) (Event, error) {
	switch event.Kind {
	case hook.KeyDown:
		key := string(event.Keychar)
		if event.Keychar == hook.CharUndefined || key == "" {
			key = hook.RawcodetoKeychar(event.Rawcode)
		}
		if key == "" {
			return Event{}, fmt.Errorf("keyboard event has no key value")
		}
		return Event{Type: Keyboard, Key: key}, nil
	case hook.MouseDown:
		return Event{Type: Mouse, Action: "down", X: int(event.X), Y: int(event.Y), Meta: map[string]string{"button": fmt.Sprint(event.Button)}}, nil
	case hook.MouseUp:
		return Event{Type: Mouse, Action: "up", X: int(event.X), Y: int(event.Y), Meta: map[string]string{"button": fmt.Sprint(event.Button)}}, nil
	case hook.MouseMove:
		return Event{Type: Mouse, Action: "move", X: int(event.X), Y: int(event.Y)}, nil
	case hook.MouseWheel:
		return Event{Type: Mouse, Action: "wheel", X: int(event.X), Y: int(event.Y), Meta: map[string]string{"amount": fmt.Sprint(event.Amount), "direction": fmt.Sprint(event.Direction)}}, nil
	default:
		return Event{}, fmt.Errorf("unsupported OS input event kind %d", event.Kind)
	}
}
