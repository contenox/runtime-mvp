package taskengine

import (
	"context"
	"time"

	"github.com/contenox/runtime-mvp/core/llmresolver"
)

// MockTaskExecutor is a mock implementation of taskengine.TaskExecutor.
type MockTaskExecutor struct {
	MockOutput       any
	MockRawResponse  string
	MockError        error
	CalledWithTask   *ChainTask
	CalledWithPrompt string

	// Add a function to dynamically return errors
	ErrorSequence []error // simulate multiple error responses
	callIndex     int
}

// TaskExec is the mock implementation of the TaskExec method.
func (m *MockTaskExecutor) TaskExec(ctx context.Context, startingTime time.Time, resolver llmresolver.Policy, currentTask *ChainTask, input any, dataType DataType) (any, DataType, string, error) {
	m.CalledWithTask = currentTask
	m.CalledWithPrompt, _ = input.(string)

	var err error
	if m.callIndex < len(m.ErrorSequence) {
		err = m.ErrorSequence[m.callIndex]
		m.callIndex++
	} else {
		err = m.MockError
	}

	return m.MockOutput, dataType, m.MockRawResponse, err
}
