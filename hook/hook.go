package hook

import (
  "github.com/phemmer/sawmill/event"
  "io"
  "fmt"
)

type Hook interface {
	Event(event *event.Event) error
}

type HookIOWriter struct {
	Output io.Writer
}
func (hook *HookIOWriter) Event(event *event.Event) (error) {
	hook.Output.Write([]byte(fmt.Sprintf("%#v\n", event)))

	return nil
}