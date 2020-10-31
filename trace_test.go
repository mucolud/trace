package trace

import (
	"context"
	"errors"
	"log"
	"testing"
	"time"
)

type MLog struct {
}

func (m MLog) Write(p []byte) (n int, err error) {
	log.Println(string(p))
	return 0, nil
}

func A(tc *TraceContext, name string, age int) {
	tc.Info(name, age)
	if err := B(tc.Trace(), name+"B", age+1); err != nil {
		_ = tc.Error("B", err.Error())
	}
	C(tc.Trace(), name+"C", age+2)
}

func B(tc *TraceContext, name string, age int) error {
	tc.Info(name, age)
	return errors.New("panic")
}

func C(tc *TraceContext, name string, age int) {
	D(tc.Trace())
	E(tc.Trace())
}

func D(tc *TraceContext) error {
	_ = tc.ErrorCustom("呵呵呵呵", "hhhh")
	return errors.New("呵呵呵呵")
}

func E(tc *TraceContext) error {
	return nil
}

func TestNewTraceContext(t *testing.T) {
	tc, _ := context.WithTimeout(context.Background(), time.Second*20)
	tc2 := NewTraceContext(tc, &MLog{})
	A(tc2, "hahah", 10)
	tc2.Log()
}

func TestTraceContext_ErrorCustom(t *testing.T) {
	tc := NewTraceContext(context.Background(), &MLog{})
	err := tc.ErrorCustom("哈哈", "笨蛋")
	t.Log(tc.WrapError(err, "你好"))
	err = tc.Error("你好")
	t.Log(tc.WrapError(err, "哈哈"))
}
