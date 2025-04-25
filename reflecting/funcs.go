package reflecting

import (
	"reflect"
	"runtime"
	"strings"
	"sync"

	commonutils "github.com/BetaGoRobot/go_utils/common_utils"
)

var pcCache = &sync.Map{}

// GetCurrentFunc 返回调用此函数的上一级函数名（经过合法化处理）
//
//	@return string
//	@update 2025-04-10 17:45:37
func GetCurrentFunc() string {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		return ""
	}

	// 尝试从缓存中获取函数名
	if cached, found := pcCache.Load(pc); found {
		return cached.(string)
	}

	// 获取函数对象
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return ""
	}

	// 解析并处理函数名
	rawName := fn.Name()
	funcName := legalize(getLastPathElement(rawName))

	// 缓存函数名
	pcCache.Store(pc, funcName)

	return funcName
}

func getLastPathElement(s string) string {
	parts := strings.Split(s, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return s
}

func legalize(s string) string {
	return commonutils.RemoveFromStringRune(s, '*', '(', ')')
}

// GetCurrentFuncDepth to be filled GetCurrentFunc 1
//
//	@param depth int
//	@return string
//	@update 2025-03-14 13:55:44
func GetCurrentFuncDepth(depth int) string {
	pc, _, _, _ := runtime.Caller(depth)
	if name, ok := pcCache.Load(pc); ok {
		return name.(string)
	}

	name, _ := pcCache.LoadOrStore(pc,
		legalize(
			getLastPathElement(runtime.FuncForPC(pc).Name()),
		),
	)
	return name.(string)
}

// GetFunctionName to be filled
//
//	@param f any
//	@return string
//	@update 2025-03-26 18:08:33
func GetFunctionName(f any) string {
	// 获取函数入口地址
	ptr := reflect.ValueOf(f).Pointer()
	fn := runtime.FuncForPC(ptr)
	if fn == nil {
		return ""
	}

	// 入口地址用于缓存键
	pc := fn.Entry()

	// 尝试从缓存获取
	if name, ok := pcCache.Load(pc); ok {
		return name.(string)
	}

	// 缓存中不存在，则处理并存储
	rawName := fn.Name()
	name := legalize(getLastPathElement(rawName))
	pcCache.Store(pc, name)

	return name
}
