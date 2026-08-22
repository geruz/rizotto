package bb

import (
	"context"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type registerTestRequest struct {
	Value string
}

type registerTestAnswer struct {
	Value  string
	Echoed bool
}

func registerTestRPC(_ context.Context, req registerTestRequest) (registerTestAnswer, ServiceError) {
	return registerTestAnswer{Value: req.Value, Echoed: true}, nil
}

func registerTestEvent(_ context.Context, _ registerTestRequest) bool {
	return true
}

func Test_MustRegister_ShouldPanicOnDuplicateURI(t *testing.T) {
	t.Parallel()

	// Arrange.
	uri := "http://bb-test-service/duplicate/rpc"
	MustRegister(uri, registerTestRPC)

	// Act, assert.
	assert.PanicsWithValue(t, "bb: RPC already registered: "+uri, func() {
		MustRegister(uri, registerTestRPC)
	})
}

func Test_MustRegisterEvent_ShouldPanicOnDuplicateURI(t *testing.T) {
	t.Parallel()

	// Arrange.
	uri := "http://bb-test-service/duplicate/event"
	MustRegisterEvent(uri, registerTestEvent)

	// Act, assert.
	assert.PanicsWithValue(t, "bb: event already registered: "+uri, func() {
		MustRegisterEvent(uri, registerTestEvent)
	})
}

func Test_MustRegister_ShouldPanicWhenURITakenByEvent(t *testing.T) {
	t.Parallel()

	// Arrange. Both kinds mount on the same routing path, so the URI is shared.
	uri := "http://bb-test-service/shared/address"
	MustRegisterEvent(uri, registerTestEvent)

	// Act, assert.
	assert.PanicsWithValue(t, "bb: event already registered: "+uri, func() {
		MustRegister(uri, registerTestRPC)
	})
}

func Test_MustRegister_ShouldBeSafeForConcurrentRegistration(t *testing.T) {
	t.Parallel()

	// Arrange.
	const registrations = 32

	var wg sync.WaitGroup

	// Act. Distinct URIs, so the only thing under test is the concurrent write.
	for i := range registrations {
		wg.Go(func() {
			MustRegister("http://bb-test-service/concurrent/"+strconv.Itoa(i), registerTestRPC)
		})
	}

	wg.Wait()

	// Assert.
	for i := range registrations {
		uri := "http://bb-test-service/concurrent/" + strconv.Itoa(i)

		handlersMu.RLock()
		_, ok := callbacks[uri]
		handlersMu.RUnlock()

		require.True(t, ok, "handler for %s went missing", uri)
	}
}

func Test_MustBind_ShouldCallRegisteredHandler(t *testing.T) {
	t.Parallel()

	// Arrange.
	uri := "http://bb-test-service/bind/call"
	MustRegister(uri, registerTestRPC)

	call := MustBind[func(context.Context, registerTestRequest) (registerTestAnswer, ServiceError)](uri)

	// Act.
	answer, err := call(t.Context(), registerTestRequest{Value: "payload"})

	// Assert.
	require.Nil(t, err)
	assert.Equal(t, "payload", answer.Value)
}

func Test_MustBind_ShouldReturnNotImplementedForUnregisteredURI(t *testing.T) {
	t.Parallel()

	// Arrange.
	call := MustBind[func(context.Context, registerTestRequest) (registerTestAnswer, ServiceError)](
		"http://bb-test-service/bind/missing",
	)

	// Act.
	_, err := call(t.Context(), registerTestRequest{Value: "payload"})

	// Assert.
	var notImplemented *NotImplementedError

	require.ErrorAs(t, err, &notImplemented)
}
