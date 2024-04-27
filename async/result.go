package async

import "context"

func Parallel2[T1 any, T2 any](
	f1 func(ctx context.Context) Result[T1],
	f2 func(ctx context.Context) Result[T2],
) func(ctx context.Context) (T1, T2, error) {
	return func(ctx context.Context) (T1, T2, error) {
		var (
			v1  T1
			v2  T2
			err error
		)

		r1 := f1(ctx)
		r2 := f2(ctx)

		select {
		case v1 = <-r1.Data:
		case e := <-r1.Err:
			err = e
		}

		select {
		case v2 = <-r2.Data:
		case e := <-r2.Err:
			err = e
		}

		return v1, v2, err
	}
}

func NewResult[T1 any](f func(success chan<- T1, err chan<- error)) Result[T1] {
	success := make(chan T1)

	err := make(chan error)
	go f(success, err)

	return Result[T1]{
		Data: success,
		Err:  err,
	}
}

func Async[T1 any](f func(ctx context.Context) (T1, error), arg T1) func(ctx context.Context) Result[T1] {
	return func(ctx context.Context) Result[T1] {
		return NewResult(func(success chan<- T1, err chan<- error) {
			v, e := f(ctx)
			if e != nil {
				err <- e
			} else {
				success <- v
			}
		})
	}
}

func Async1[T1 any, TReq1 any](
	f func(context.Context, TReq1) (T1, error),
	arg TReq1,
) func(ctx context.Context) Result[T1] {
	return func(ctx context.Context) Result[T1] {
		return NewResult(func(success chan<- T1, err chan<- error) {
			v, e := f(ctx, arg)
			if e != nil {
				err <- e
			} else {
				success <- v
			}
		})
	}
}

func Async2[T1 any, TReq1 any, TReq2 any](
	f func(context.Context, TReq1, TReq2) (T1, error),
	arg1 TReq1,
	arg2 TReq2,
) func(ctx context.Context) Result[T1] {
	return func(ctx context.Context) Result[T1] {
		return NewResult(func(success chan<- T1, err chan<- error) {
			v, e := f(ctx, arg1, arg2)
			if e != nil {
				err <- e
			} else {
				success <- v
			}
		})
	}
}

type Result[Data any] struct {
	Data <-chan Data
	Err  <-chan error
}
