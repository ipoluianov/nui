//go:build linux
// +build linux

#include "ximage_helper.h"

void destroy_ximage(XImage* img) {
    XDestroyImage(img);
}
