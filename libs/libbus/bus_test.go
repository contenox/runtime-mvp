package libbus_test

import (
	"context"
	"testing"
	"time"

	"github.com/contenox/runtime-mvp/libs/libbus"
	"github.com/stretchr/testify/require"
)

func TestStream(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ps, cleanup, err := libbus.NewTestPubSub()
	defer cleanup()
	if err != nil {
		t.Fatalf("failed to init test stream %s", err)
	}

	subject := "test.stream"
	message := []byte("streamed message")

	// Create a channel for streaming messages.
	streamCh := make(chan []byte, 1)
	sub, err := ps.Stream(ctx, subject, streamCh)
	require.NoError(t, err)
	defer sub.Unsubscribe()

	// Publish the message.
	err = ps.Publish(ctx, subject, message)
	require.NoError(t, err)

	// Wait for the streamed message.
	select {
	case received := <-streamCh:
		require.Equal(t, message, received)
	case <-ctx.Done():
		t.Fatal("timed out waiting for streamed message")
	}
}

func TestPublishWithClosedConnection(t *testing.T) {
	ctx := context.Background()

	ps, cleanup, err := libbus.NewTestPubSub()
	defer cleanup()
	if err != nil {
		t.Fatalf("failed to init test stream %s", err)
	}
	// Close the connection.
	err = ps.Close()
	require.NoError(t, err)

	// Attempt to publish after closing.
	err = ps.Publish(ctx, "test.closed", []byte("data"))
	require.Error(t, err)
	require.Equal(t, libbus.ErrConnectionClosed, err)
}
