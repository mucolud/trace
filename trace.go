package trace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/mucolud/lib/convert"
)

const (
	colorRed    = 31
	colorYellow = 33
)

var customError = errors.New("custom error: ")

type node struct {
	File string        `json:"file"`
	Func string        `json:"func"`
	Data []interface{} `json:"data"`
}
type TraceContext struct {
	context.Context
	traceId  int64
	mux      sync.Mutex
	logger   io.Writer
	funcName string
	errors   []*node
	infos    []*node
	children []*TraceContext
}

func NewTraceContext(ctx context.Context, logger io.Writer) *TraceContext {
	pc, _, _, _ := runtime.Caller(1)
	funcName := ""
	if pcFunc := runtime.FuncForPC(pc); pcFunc != nil {
		funcName = pcFunc.Name()
	}
	return &TraceContext{
		Context:  ctx,
		logger:   logger,
		traceId:  time.Now().UnixNano(),
		funcName: funcName,
		children: make([]*TraceContext, 0, 10),
		errors:   make([]*node, 0, 10),
		infos:    make([]*node, 0, 10),
	}
}

func withColor(color int, str interface{}) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", color, convert.ToString(str))
}

func (tc *TraceContext) convertParams(params []interface{}) []interface{} {
	for i, v := range params {
		if err, ok := v.(error); ok && err != nil {
			params[i] = err.Error()
		}
	}
	return params
}

func (tc *TraceContext) convertToError(params []interface{}) error {
	if len(params) == 0 {
		return nil
	}

	var res = make([]string, 0, len(params))
	for _, v := range params {
		if err, ok := v.(error); ok && err != nil {
			res = append(res, err.Error())
		} else {
			if v == nil {
				res = append(res, "nil")
			} else {
				res = append(res, fmt.Sprintf("%+v",
					reflect.Indirect(reflect.ValueOf(v)).Interface()))
			}
		}
	}
	return errors.New(strings.Join(res, ","))
}

func (tc *TraceContext) Trace() *TraceContext {
	tc.mux.Lock()
	defer tc.mux.Unlock()
	if tc.children == nil {
		tc.children = make([]*TraceContext, 0, 10)
	}
	pc, _, _, _ := runtime.Caller(1)
	funcName := ""
	if pcFunc := runtime.FuncForPC(pc); pcFunc != nil {
		funcName = pcFunc.Name()
	}
	ntc := NewTraceContext(tc.Context, tc.logger)
	ntc.traceId = tc.traceId
	ntc.funcName = funcName
	tc.children = append(tc.children, ntc)
	return ntc
}

func (tc *TraceContext) Error(params ...interface{}) error {
	pc, _, line, _ := runtime.Caller(1)
	funcName := ""
	if pcFunc := runtime.FuncForPC(pc); pcFunc != nil {
		funcName = pcFunc.Name()
	}
	tc.errors = append(tc.errors, &node{
		File: fmt.Sprintf("%d", line),
		Func: funcName,
		Data: tc.convertParams(params),
	})

	return tc.convertToError(params)
}

func (tc *TraceContext) ErrorCustom(params ...interface{}) error {
	ve := tc.convertToError(params)
	if ve == nil {
		return nil
	} else {
		return fmt.Errorf("%w%s", customError, ve.Error())
	}
}

func (tc *TraceContext) WrapError(err error, title ...string) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, customError) {
		return errors.New(strings.ReplaceAll(err.Error(), customError.Error(), ""))
	} else {
		if len(title) > 0 {
			return errors.New(strings.Join(title, ","))
		} else {
			return err
		}
	}
}

func (tc *TraceContext) Info(params ...interface{}) {
	pc, _, line, _ := runtime.Caller(1)
	funcName := ""
	if pcFunc := runtime.FuncForPC(pc); pcFunc != nil {
		funcName = pcFunc.Name()
	}
	tc.infos = append(tc.infos, &node{
		File: fmt.Sprintf("%d", line),
		Func: funcName,
		Data: tc.convertParams(params),
	})
}

func (tc *TraceContext) formatLog(node *TraceContext, prefix string) string {
	//┌ ┬ ┐
	//├ ┼ ┤
	//└ ┴ ┘

	var str = &strings.Builder{}
	//var hasLog = len(node.infos) > 0 && len(node.errors) > 0
	str.WriteString(node.funcName + "\n")

	for _, v := range node.infos {
		infoStr := ""
		if len(v.Data) > 0 {
			res, _ := json.Marshal(v.Data)
			infoStr = strings.ReplaceAll(string(res), "\\", "") + "\n"
		}
		str.WriteString(prefix + "├> " + v.Func + ":" + v.File + ":" + infoStr)
	}
	for _, v := range node.errors {
		infoStr := ""
		if len(v.Data) > 0 {
			res, _ := json.Marshal(v.Data)
			infoStr = string(res) + "\n"
		}
		str.WriteString(prefix + "├E " + v.Func + ":" + v.File + ":" + infoStr)
	}

	if len(node.children) > 0 {
		for _, v := range node.children {
			if len(v.errors) == 0 && len(v.infos) == 0 && len(v.children) == 0 {
				continue
			}
			tag := "├"
			outLog := tc.formatLog(v, prefix+"   ")
			if outLog != "" {
				str.WriteString(prefix + tag + outLog)
			}
		}
	}
	return str.String()
}

func (tc *TraceContext) Log() {
	if tc.logger != nil {
		split := fmt.Sprintf("traceId:%d", tc.traceId)
		tc.logger.Write([]byte(
			withColor(colorYellow, "\n\n┌ "+split+"\n") +
				tc.formatLog(tc, "") +
				withColor(colorYellow, "└ "+split),
		))
	}
}
