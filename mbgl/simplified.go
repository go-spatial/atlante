package mbgl

/*
#cgo CFLAGS: -fPIC
#cgo CFLAGS: -D_GLIBCXX_USE_CXX11_ABI=1
#cgo CXXFLAGS: -std=c++14 -std=gnu++14
#cgo CXXFLAGS: -g
*/
import "C"

var runLoop *cRunLoop

func NewRunLoop() {
	if runLoop == nil {
		runLoop = newCRunLoop()
	}
}

func DestroyRunLoop() {
	if runLoop != nil {
		runLoop.Destruct()
		runLoop = nil
	}
}
