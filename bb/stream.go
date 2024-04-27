package bb

/*
import (
	"context"
	"errors"
	"net/url"
	"os"
	"strings"

)

var (
	streams               = map[string]*streamConfig{}
	ErrWrongParamsType    = errors.New("wrong subscription params")
	ErrProviderIsNull     = errors.New("provider is nil")
	ErrProviderHasNotInit = errors.New("current provider has not init")
)

type (
	Stream[TEvent any]          func(ctx context.Context) (<-chan TEvent, func(), error)
	StreamPublisher[TEvent any] interface {
		Publish(ctx context.Context, event TEvent) error
	}
)

type streamConfig struct {
	makeProvider func(ctx context.Context, uri string, consumer *rabbitmq.Consumer) interface{}
	curProvider  interface{}
}

type subscribers[T any] struct {
	ch chan<- T
}

type subscribeChanProvider[T any] struct {
	broadcast    chan T
	addListen    chan subscribers[T]
	removeListen chan T
	listeners    []subscribers[T]
}

func (p *subscribeChanProvider[T]) Subscribe() (<-chan T, func()) {
	ch := make(chan T, 10)
	p.addListen <- (subscribers[T]{
		ch: ch,
	})
	return ch, func() {
		p.removeListen <- ch
	}

func (p *subscribeChanProvider[T]) loop(ctx context.Context) {
	for {
		select {
		case listener := <-p.addListen:
			{
				p.listeners = append(p.listeners, listener)
			}
		case ch := <-p.removeListen:
			{
				for i, l := range p.listeners {
					if l.ch == ch {
						p.listeners = append(p.listeners[:i], p.listeners[i+1:]...)
						break
					}
		case data := <-p.broadcast:
			{
				for _, listener := range p.listeners {
					select {
					case listener.ch <- data:
					default:
					}
		case <-ctx.Done():
			return
		}

func MustBindSubscribe[TEvent any](
	uri string,
) Stream[TEvent] {
	p, ok := streams[uri]
	if !ok {
		p = mustBindClient[TEvent](uri)
		streams[uri] = p
	}
	return func(ctx context.Context) (<-chan TEvent, func(), error) {
		curProvider := p.curProvider
		if curProvider == nil {
			return nil, ErrProviderIsNull
		}
		provider, ok := curProvider.(subscribeChanProvider[TEvent])
		if !ok {
			return nil, ErrProviderHasNotInit
		}
		ch, unsubscribe := provider.Subscribe()
		return ch, unsubscribe, nil
	}

func StartSubscriptionsTransport(ctx context.Context, getConnection rabbitmq.ConnectionProvider) {
	logger.Info(ctx, "Start subscriptions transport")
	for uri, handlerParams := range streams {
		logger.Info(ctx, "Start subscription for "+uri)
		p := mustParseURI(uri)

		consumer := rabbitmq.NewConsumer(
			getConnection,
			p.exchange,
			p.queue,
			"*",
		)
		consumer.SetExchangeType("fanout")
		// consumer.SetExclusive(true)
		consumer.SetAutoDelete(true)
		consumer.SetNoWait(true)
		consumer.SetAutoAck(true)
		consumer.SetUseDlx(false)
		handlerParams.curProvider = handlerParams.makeProvider(ctx, uri, consumer)
		streams[uri] = handlerParams
	}

func newStreamConfig[TEvent any]() *streamConfig {
	return &streamConfig{
		curProvider: nil,
		makeProvider: func(ctx context.Context, uri string, consumer *rabbitmq.Consumer) interface{} {
			provider := subscribeChanProvider[TEvent]{
				broadcast:    make(chan TEvent, 10),
				listeners:    []subscribers[TEvent]{},
				addListen:    make(chan subscribers[TEvent]),
				removeListen: make(chan chan TEvent),
			}
			go func() {
				for {
					func() {
						utils.Recovery(ctx)
						provider.loop(ctx)
					}()
				}
			}()
			consumer.Consume(ctx,
				func() interface{} { return new(TEvent) },
				func(cxt context.Context, data interface{}) bool {
					if model, ok := data.(*TEvent); ok {
						provider.broadcast <- *model
						return true
					} else {
						logger.Error(ctx, "Failed to cast data to "+uri, nil)
					}

					return true
				}, 1)
			return provider
		},
	}

func MustRegisterStreamPublisher[TEvent any](
	uri string, getConnection rabbitmq.ConnectionProvider,
) StreamPublisher[TEvent] {
	p := mustParseURI(uri)
	publisher := rabbitmq.NewPublisher[TEvent](
		getConnection,
		p.exchange,
		"fanout",
		"*",
	)
	return publisher
}

type rabbitMQparams struct {
	exchange string
	queue    string
}

func mustParseURI(uri string) rabbitMQparams {
	parsedURL, err := url.Parse(uri)
	if err != nil {
		panic(err)
	}
	exchange := strings.ReplaceAll(parsedURL.Path, "/", "")
	host, _ := os.Hostname()
	return rabbitMQparams{
		exchange: exchange,
		queue:    exchange + "_" + host + "_" + uuid.New().String(),
	}

func mustBindClient[TEvent any](uri string) *streamConfig {
	scheme := mustGetScheme(uri)
	switch scheme {
	case rabbitProtocol:
		return newStreamConfig[TEvent]()
	default:
		panic("Unknown scheme for subscription: " + scheme)
	}
*/
