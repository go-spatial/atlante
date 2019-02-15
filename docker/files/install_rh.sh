#!/bin/bash -x


# copy includes will use find and xargs to copy includes from the mason_packages/headers folder
# to the $PKG_ROOT/include directory
# find "mapbox-gl-native/mason_packages/headers/geometry" -iname include | xargs -n 1 -I {} echo "DIR id {} stuff"
function copy_includes {
	subdir=$1
	shift
	for libraryName in $@; do
	  srcdir="${PKG_ROOT}/mapbox-gl-native/${subdir}/${libName}"
	  echo "copying includes for ${PKG_ROOT}/mapbox-gl-native/${subdir}/${libraryName}"
 	  find "${PKG_ROOT}/mapbox-gl-native/${subdir}/${libraryName}" -iname include | xargs -n 1 -I{} cp -R {}/ ${PKG_ROOT}/
	done	
}

function copy_hpps {
	subdir=$1
	shift
	echo "remainder $@"
	for dir in $@; do
	  srcdir="${PKG_ROOT}/mapbox-gl-native/${subdir}/${dir}"
	  for file in $(find ${srcdir} -type f -name '*.hpp'); do
		destfile=${file#$srcdir}
		bdir=$(dirname ${destfile})
		echo copying $(basename $file) " to ${INCLUDEDIR}/$destfile"
		mkdir -p ${INCLUDEDIR}/${bdir}
		cp $file ${INCLUDEDIR}/${destfile}
	   done
	done	
}

function copy_libs {
      subdir=$1
      shift
      for libName in $@; do 
	srcdir="${PKG_ROOT}/mapbox-gl-native/${subdir}/${osdir}/${libName}"
	for afile in $( find "${srcdir}" -iname "*.a" ); do
		cp -R ${afile} ${LIBDIR}/
	done
      done
}

unamestr=`uname`
echo "running on $(uname)"


if [[ ! $GOPATH ]]; then
    echo "GOPATH must be set"
    exit
fi

PKG_ROOT=$GOPATH/src/github.com/go-spatial/go-mbgl/mbgl/c
osdir="linux-x86_64"


# download and install sdk
if [[ ! -d mapbox-gl-native ]]; then
    git clone https://github.com/mapbox/mapbox-gl-native
	 pushd mapbox-gl-native
	 git config user.email "gautam.dey77@gmail.com"
	 git config user.name "Gautam Dey"
	 git checkout 98eac18a2133a7beda12fdfc27d6f88217d800cf
	 git reset --hard 
	 git submodule init
	 git submodule update
	 git apply ../patches/*

	 popd

fi

if [[ ! -d $PKG_ROOT/lib ]]; then
    mkdir $PKG_ROOT/lib
fi

cd $PKG_ROOT/mapbox-gl-native

LIBDIR=$PKG_ROOT/lib/linux
INCLUDEDIR=$PKG_ROOT/include

make WITH_OSMESA=ON linux-core

if [[ -d ${LIBDIR} ]]; then
    rm -rf ${LIBDIR}/*.a
    rm -rf ${INCLUDEDIR}
fi

mkdir -p ${LIBDIR}
mkdir -p ${INCLUDEDIR}

copy_libs "build" "Debug"
copy_libs "mason_packages" "libuv" "libjpeg-turbo" "libpng"

#cp $PKG_ROOT/mapbox-gl-native/build/linux-x86_64/Debug/*.a ${LIBDIR}
cp -R $PKG_ROOT/mapbox-gl-native/include/* ${INCLUDEDIR}

copy_hpps "platform" "default"

copy_includes "vendor" "expected" "geometry" "variant"

cp -R $PKG_ROOT/mapbox-gl-native/vendor/geometry.hpp/include/* ${INCLUDEDIR}

