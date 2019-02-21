package mbgl

/*
#cgo CXXFLAGS: -I${SRCDIR}/c/include
#cgo CXXFLAGS: -I${SRCDIR}/c/include/include
#cgo LDFLAGS: -L${SRCDIR}/c/lib/linux

#cgo LDFLAGS: ${SRCDIR}/c/lib/linux/libmbgl-core.a
#cgo LDFLAGS: ${SRCDIR}/c/lib/linux/libmbgl-filesource.a
#cgo LDFLAGS: ${SRCDIR}/c/lib/linux/libmbgl-loop-uv.a
#cgo LDFLAGS: ${SRCDIR}/c/lib/linux/libsqlite.a

#cgo LDFLAGS: -Wl,--no-as-needed -lcurl
#cgo LDFLAGS: -Wl,--as-needed ${SRCDIR}/c/lib/linux/libmbgl-core.a
#cgo LDFLAGS: -lOSMesa
#cgo LDFLAGS: -lz -lm
#cgo LDFLAGS: ${SRCDIR}/c/lib/linux/libnunicode.a
#cgo LDFLAGS: ${SRCDIR}/c/lib/linux/libicu.a
#cgo LDFLAGS: ${SRCDIR}/c/lib/linux/libuv.a
#cgo LDFLAGS: -lrt -lpthread
#cgo LDFLAGS: -lnsl -ldl
#cgo LDFLAGS: -static-libstdc++
#cgo pkg-config: libpng libjpeg

*/
import "C"

/*
LDFLAGS: ${SRCDIR}/c/lib/linux/libpng.a
#cgo LDFLAGS: ${SRCDIR}/c/lib/linux/libjpeg.a
*/
