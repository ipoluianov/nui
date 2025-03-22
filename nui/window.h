#ifndef WINDOW_H
#define WINDOW_H

int InitWindow(void);
void RunEventLoop(void);

void go_on_paint(void* buffer, int width, int height, int hwnd);
void go_on_key_down(int keycode);
void go_on_key_up(int keycode);
void go_on_modifier_change(int shift, int ctrl, int alt, int cmd);
void go_on_char(int codepoint);

#endif
