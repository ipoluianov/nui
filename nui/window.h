#ifndef WINDOW_H
#define WINDOW_H

int InitWindow(void);
void RunEventLoop(void);

void CloseWindowById(int windowId);
void SetWindowTitle(int windowId, const char* title);
void SetWindowSize(int windowId, int width, int height);
void SetWindowPosition(int windowId, int x, int y);
void MinimizeWindow(int windowId);
void MaximizeWindow(int windowId);



void go_on_paint(void* buffer, int width, int height, int hwnd);
void go_on_key_down(int keycode);
void go_on_key_up(int keycode);
void go_on_modifier_change(int shift, int ctrl, int alt, int cmd);
void go_on_char(int codepoint);

void go_on_mouse_down(int button, int x, int y);
void go_on_mouse_up(int button, int x, int y);
void go_on_mouse_move(int x, int y);
void go_on_mouse_scroll(int delta);
void go_on_mouse_enter(void);
void go_on_mouse_leave(void);
void go_on_mouse_double_click(int button, int x, int y);


#endif
