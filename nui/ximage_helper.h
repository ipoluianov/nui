//go:build linux
// +build linux

#pragma once
#include <X11/Xlib.h>

void destroy_ximage(XImage* img);
