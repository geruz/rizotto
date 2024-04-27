package mock

import (
	"context"
	"strconv"

	"github.com/geruz/rizotto/bb"
)

type CallParams[TRequest any, TResponse any] struct {
	Request  TRequest
	Response TResponse
	err      error
	ctx      context.Context //nolint
}
type MockConfigurator[TRequest any, TResponse any] struct {
	response TResponse
	err      bb.ServiceError
	calls    []CallParams[TRequest, TResponse]
}

func (m *MockConfigurator[TRequest, TResponse]) call(_ context.Context, _r TRequest) (TResponse, bb.ServiceError) {
	return m.response, m.err
}

func (m *MockConfigurator[TRequest, TResponse]) Bind() func(
	ctx context.Context, r TRequest,
) (TResponse, bb.ServiceError) {
	return func(ctx context.Context, r TRequest) (TResponse, bb.ServiceError) {
		res, err := m.call(ctx, r)
		m.calls = append(m.calls, CallParams[TRequest, TResponse]{
			Request:  r,
			Response: res,
			err:      err,
			ctx:      ctx,
		})

		return res, err
	}
}

func (m *MockConfigurator[TRequest, TResponse]) Success(resp TResponse) *MockConfigurator[TRequest, TResponse] {
	m.response = resp

	return m
}

func (m *MockConfigurator[TRequest, TResponse]) Error(err bb.ServiceError) *MockConfigurator[TRequest, TResponse] {
	m.err = err

	return m
}

func (m *MockConfigurator[TRequest, TResponse]) CallsCount() int {
	return len(m.calls)
}

func (m *MockConfigurator[TRequest, TResponse]) Call(n int) CallParams[TRequest, TResponse] {
	if n >= len(m.calls) {
		panic("Count of calls is less than expected, expected: " + strconv.Itoa(n) +
			", actual: " + strconv.Itoa(len(m.calls)))
	}

	return m.calls[n]
}

func (m *MockConfigurator[TRequest, TResponse]) ExpectedCalls(int) *MockConfigurator[TRequest, TResponse] {
	return m
}

func (m *MockConfigurator[TRequest, TResponse]) ExpectedRequest(
	callNumber int,
	expected TRequest,
) *MockConfigurator[TRequest, TResponse] {
	return m
}

func RPC[
	F ~func(ctx context.Context, req TRequest) (TResponse, bb.ServiceError),
	TRequest any,
	TResponse any,
](
	f *F,
) *MockConfigurator[TRequest, TResponse] {
	conf := MockConfigurator[TRequest, TResponse]{
		response: *new(TResponse),
		err:      nil,
		calls:    []CallParams[TRequest, TResponse]{},
	}
	*f = conf.Bind()

	return &conf
}
