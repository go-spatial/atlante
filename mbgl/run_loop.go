package mbgl

/*
#include "run_loop.h"
*/
import "C"

type cRunLoop C.MbglRunLoop

func newCRunLoop() *cRunLoop {
	ptr := C.mbgl_run_loop_new()

	return (*cRunLoop)(ptr)
}

func (rl *cRunLoop) Destruct() {
	C.mbgl_run_loop_destruct((*C.MbglRunLoop)(rl))
}
