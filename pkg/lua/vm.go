package lua

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/Shopify/go-lua"
)

type VM struct {
	state     *lua.State
	timers    map[int]*Timer
	timerID   int
	timerLock sync.Mutex
}

type Timer struct {
	ID       int
	Callback string
	Interval time.Duration
	Repeat   bool
	NextRun  time.Time
	Args     []interface{}
}

func NewVM() *VM {
	state := lua.NewState()
	openSafeLibraries(state)
	return &VM{
		state:  state,
		timers: make(map[int]*Timer),
	}
}

func openSafeLibraries(state *lua.State) {
	lua.OpenLibraries(state)

	state.PushNil()
	state.SetGlobal("io")

	state.PushNil()
	state.SetGlobal("os")

	state.PushNil()
	state.SetGlobal("debug")

	state.PushNil()
	state.SetGlobal("dofile")

	state.PushNil()
	state.SetGlobal("loadfile")
}

func (vm *VM) LoadFile(path string) error {
	if err := lua.DoFile(vm.state, path); err != nil {
		return fmt.Errorf("failed to load lua file %s: %w", path, err)
	}
	return nil
}

func (vm *VM) LoadString(code string) error {
	if err := lua.DoString(vm.state, code); err != nil {
		return fmt.Errorf("failed to load lua string: %w", err)
	}
	return nil
}

func (vm *VM) Close() {
	vm.timerLock.Lock()
	vm.timers = make(map[int]*Timer)
	vm.timerLock.Unlock()
}

func (vm *VM) RegisterTimer(callback string, interval time.Duration, repeat bool, args ...interface{}) int {
	vm.timerLock.Lock()
	defer vm.timerLock.Unlock()

	vm.timerID++
	timer := &Timer{
		ID:       vm.timerID,
		Callback: callback,
		Interval: interval,
		Repeat:   repeat,
		NextRun:  time.Now().Add(interval),
		Args:     args,
	}

	vm.timers[timer.ID] = timer
	return timer.ID
}

func (vm *VM) CancelTimer(id int) {
	vm.timerLock.Lock()
	defer vm.timerLock.Unlock()

	delete(vm.timers, id)
}

func (vm *VM) UpdateTimers() error {
	vm.timerLock.Lock()
	now := time.Now()
	var toExecute []*Timer
	var toRemove []int

	for _, timer := range vm.timers {
		if now.After(timer.NextRun) || now.Equal(timer.NextRun) {
			toExecute = append(toExecute, timer)
			if timer.Repeat {
				timer.NextRun = now.Add(timer.Interval)
			} else {
				toRemove = append(toRemove, timer.ID)
			}
		}
	}

	for _, id := range toRemove {
		delete(vm.timers, id)
	}
	vm.timerLock.Unlock()

	for _, timer := range toExecute {
		if err := vm.CallFunction(timer.Callback, timer.Args...); err != nil {
			return fmt.Errorf("timer callback %s failed: %w", timer.Callback, err)
		}
	}

	return nil
}

func (vm *VM) GetGlobalString(name string) (string, error) {
	vm.state.Global(name)
	if !vm.state.IsString(-1) {
		vm.state.Pop(1)
		return "", fmt.Errorf("global %s is not a string", name)
	}
	value, _ := vm.state.ToString(-1)
	vm.state.Pop(1)
	return value, nil
}

func (vm *VM) GetGlobalNumber(name string) (float64, error) {
	vm.state.Global(name)
	if !vm.state.IsNumber(-1) {
		vm.state.Pop(1)
		return 0, fmt.Errorf("global %s is not a number", name)
	}
	value, _ := vm.state.ToNumber(-1)
	vm.state.Pop(1)
	return value, nil
}

func (vm *VM) GetGlobalBool(name string) (bool, error) {
	vm.state.Global(name)
	if !vm.state.IsBoolean(-1) {
		vm.state.Pop(1)
		return false, fmt.Errorf("global %s is not a boolean", name)
	}
	value := vm.state.ToBoolean(-1)
	vm.state.Pop(1)
	return value, nil
}

func (vm *VM) GetGlobalTable(name string) error {
	vm.state.Global(name)
	if !vm.state.IsTable(-1) {
		vm.state.Pop(1)
		return fmt.Errorf("global %s is not a table", name)
	}
	return nil
}

func (vm *VM) GetTableString(key string) (string, error) {
	vm.state.Field(-1, key)
	if !vm.state.IsString(-1) {
		vm.state.Pop(1)
		return "", fmt.Errorf("field %s is not a string", key)
	}
	value, _ := vm.state.ToString(-1)
	vm.state.Pop(1)
	return value, nil
}

func (vm *VM) GetTableNumber(key string) (float64, error) {
	vm.state.Field(-1, key)
	if !vm.state.IsNumber(-1) {
		vm.state.Pop(1)
		return 0, fmt.Errorf("field %s is not a number", key)
	}
	value, _ := vm.state.ToNumber(-1)
	vm.state.Pop(1)
	return value, nil
}

func (vm *VM) GetTableBool(key string) (bool, error) {
	vm.state.Field(-1, key)
	if !vm.state.IsBoolean(-1) {
		vm.state.Pop(1)
		return false, fmt.Errorf("field %s is not a boolean", key)
	}
	value := vm.state.ToBoolean(-1)
	vm.state.Pop(1)
	return value, nil
}

func (vm *VM) GetTableIntArray(key string) ([]int, error) {
	vm.state.Field(-1, key)
	if !vm.state.IsTable(-1) {
		vm.state.Pop(1)
		return nil, fmt.Errorf("field %s is not a table", key)
	}

	var result []int
	length := vm.state.RawLength(-1)
	for i := 1; i <= length; i++ {
		vm.state.RawGetInt(-1, i)
		if !vm.state.IsNumber(-1) {
			vm.state.Pop(2)
			return nil, fmt.Errorf("field %s[%d] is not a number", key, i)
		}
		value, _ := vm.state.ToNumber(-1)
		result = append(result, int(value))
		vm.state.Pop(1)
	}
	vm.state.Pop(1)
	return result, nil
}

func (vm *VM) PopTable() {
	vm.state.Pop(1)
}

func (vm *VM) CallFunction(name string, args ...interface{}) error {
	vm.state.Global(name)
	if !vm.state.IsFunction(-1) {
		vm.state.Pop(1)
		return fmt.Errorf("global %s is not a function", name)
	}

	for _, arg := range args {
		switch v := arg.(type) {
		case string:
			vm.state.PushString(v)
		case int:
			vm.state.PushInteger(v)
		case float64:
			vm.state.PushNumber(v)
		case bool:
			vm.state.PushBoolean(v)
		default:
			vm.state.Pop(1)
			return fmt.Errorf("unsupported argument type: %T", arg)
		}
	}

	if err := vm.state.ProtectedCall(len(args), 0, 0); err != nil {
		return vm.enhanceError(fmt.Sprintf("function %s", name), err)
	}

	return nil
}

func (vm *VM) enhanceError(context string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("[Lua Error] %s: %w", context, err)
}

func (vm *VM) CallFunctionWithReturn(name string, numReturns int, args ...interface{}) ([]interface{}, error) {
	vm.state.Global(name)
	if !vm.state.IsFunction(-1) {
		vm.state.Pop(1)
		return nil, fmt.Errorf("global %s is not a function", name)
	}

	for _, arg := range args {
		switch v := arg.(type) {
		case string:
			vm.state.PushString(v)
		case int:
			vm.state.PushInteger(v)
		case float64:
			vm.state.PushNumber(v)
		case bool:
			vm.state.PushBoolean(v)
		default:
			vm.state.Pop(1)
			return nil, fmt.Errorf("unsupported argument type: %T", arg)
		}
	}

	if err := vm.state.ProtectedCall(len(args), numReturns, 0); err != nil {
		return nil, fmt.Errorf("failed to call function %s: %w", name, err)
	}

	results := make([]interface{}, numReturns)
	for i := numReturns - 1; i >= 0; i-- {
		stackIndex := -1 - (numReturns - 1 - i)
		switch {
		case vm.state.IsString(stackIndex):
			value, _ := vm.state.ToString(stackIndex)
			results[i] = value
		case vm.state.IsNumber(stackIndex):
			value, _ := vm.state.ToNumber(stackIndex)
			results[i] = value
		case vm.state.IsBoolean(stackIndex):
			value := vm.state.ToBoolean(stackIndex)
			results[i] = value
		case vm.state.IsNil(stackIndex):
			results[i] = nil
		default:
			results[i] = nil
		}
	}
	vm.state.Pop(numReturns)

	return results, nil
}

func (vm *VM) HasFunction(name string) bool {
	vm.state.Global(name)
	isFunc := vm.state.IsFunction(-1)
	vm.state.Pop(1)
	return isFunc
}

func (vm *VM) RegisterFunction(name string, fn lua.Function) {
	vm.state.Register(name, fn)
}

func (vm *VM) State() *lua.State {
	return vm.state
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
