#include "ximage_helper.h"

void destroy_ximage(XImage* img) {
    XDestroyImage(img);
}
