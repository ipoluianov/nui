//go:build linux
// +build linux

#include <X11/Xlib.h>
#include <X11/Xatom.h>
#include "ximage_helper.h"

void destroy_ximage(XImage* img) {
    XDestroyImage(img);
}

void maximizeWindow(Display* display, Window window) {
    Atom wmState = XInternAtom(display, "_NET_WM_STATE", False);
    Atom maxH = XInternAtom(display, "_NET_WM_STATE_MAXIMIZED_HORZ", False);
    Atom maxV = XInternAtom(display, "_NET_WM_STATE_MAXIMIZED_VERT", False);

    XClientMessageEvent xev = {0};
    xev.type = ClientMessage;
    xev.serial = 0;
    xev.send_event = True;
    xev.window = window;
    xev.message_type = wmState;
    xev.format = 32;
    xev.data.l[0] = 1; // _NET_WM_STATE_ADD
    xev.data.l[1] = maxH;
    xev.data.l[2] = maxV;
    xev.data.l[3] = 0;
    xev.data.l[4] = 0;

    Window root = DefaultRootWindow(display);
    XSendEvent(display, root, False, SubstructureRedirectMask | SubstructureNotifyMask, (XEvent*)&xev);
}

void restoreWindow(Display* display, Window window) {
    Atom wmState = XInternAtom(display, "_NET_WM_STATE", False);
    Atom maxH = XInternAtom(display, "_NET_WM_STATE_MAXIMIZED_HORZ", False);
    Atom maxV = XInternAtom(display, "_NET_WM_STATE_MAXIMIZED_VERT", False);

    XClientMessageEvent xev = {0};
    xev.type = ClientMessage;
    xev.serial = 0;
    xev.send_event = True;
    xev.window = window;
    xev.message_type = wmState;
    xev.format = 32;
    xev.data.l[0] = 0; // _NET_WM_STATE_REMOVE
    xev.data.l[1] = maxH;
    xev.data.l[2] = maxV;
    xev.data.l[3] = 0;
    xev.data.l[4] = 0;

    Window root = DefaultRootWindow(display);
    XSendEvent(display, root, False, SubstructureRedirectMask | SubstructureNotifyMask, (XEvent*)&xev);
}

void minimizeWindow(Display* display, Window window) {
    int screen = DefaultScreen(display);
    XIconifyWindow(display, window, screen);
}
