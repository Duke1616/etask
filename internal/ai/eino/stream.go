package eino

import (
	"errors"
	"io"
	"sync"

	internalai "github.com/Duke1616/etask/internal/ai"
	"github.com/cloudwego/eino/schema"
)

type responseStream struct {
	reader    *schema.StreamReader[*schema.Message]
	chunks    []*schema.Message
	pending   []internalai.Event
	current   internalai.Event
	err       error
	finished  bool
	release   sync.Once
	closeOnce sync.Once
	close     func()
}

func newResponseStream(reader *schema.StreamReader[*schema.Message], close func()) *responseStream {
	return &responseStream{reader: reader, close: close}
}

func (s *responseStream) Next() bool {
	if len(s.pending) > 0 {
		s.current, s.pending = s.pending[0], s.pending[1:]
		return true
	}
	if s.finished {
		return false
	}
	for {
		message, err := s.reader.Recv()
		if errors.Is(err, io.EOF) {
			s.finish()
			return s.Next()
		}
		if err != nil {
			s.err = err
			s.finished = true
			s.releaseResources()
			return false
		}
		if message == nil {
			continue
		}
		s.chunks = append(s.chunks, message)
		if message.Content != "" {
			s.current = internalai.Event{Type: internalai.EventTypeTextDelta, Text: message.Content}
			return true
		}
	}
}

func (s *responseStream) finish() {
	s.finished = true
	if len(s.chunks) == 0 {
		s.pending = append(s.pending, internalai.Event{Type: internalai.EventTypeCompleted})
		s.releaseResources()
		return
	}
	message, err := schema.ConcatMessages(s.chunks)
	if err != nil {
		s.err = err
		s.releaseResources()
		return
	}
	for _, toolCall := range message.ToolCalls {
		s.pending = append(s.pending, internalai.Event{
			Type: internalai.EventTypeToolCall,
			ToolCall: &internalai.ToolCall{
				Name: toolCall.Function.Name, Arguments: toolCall.Function.Arguments,
			},
		})
	}
	completed := internalai.Event{Type: internalai.EventTypeCompleted}
	if message.ResponseMeta != nil && message.ResponseMeta.Usage != nil {
		completed.Usage = internalai.Usage{
			InputTokens:  int64(message.ResponseMeta.Usage.PromptTokens),
			OutputTokens: int64(message.ResponseMeta.Usage.CompletionTokens),
		}
	}
	s.pending = append(s.pending, completed)
	s.releaseResources()
}

func (s *responseStream) Current() internalai.Event { return s.current }
func (s *responseStream) Err() error                { return s.err }

func (s *responseStream) Close() error {
	s.closeOnce.Do(s.reader.Close)
	s.releaseResources()
	return nil
}

func (s *responseStream) releaseResources() {
	s.release.Do(func() {
		if s.close != nil {
			s.close()
		}
	})
}
