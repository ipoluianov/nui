#ifndef WINDOW_H
#define WINDOW_H

int InitWindow(void);
void RunEventLoop(void);

void go_on_paint(void* buffer, int width, int height, int hwnd);

#endif
