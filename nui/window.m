#import <Cocoa/Cocoa.h>
#import "window.h"

static NSWindow* window;

@interface AppDelegate : NSObject <NSApplicationDelegate>
@end

@implementation AppDelegate

- (BOOL)applicationShouldTerminateAfterLastWindowClosed:(NSApplication *)sender {
    return YES;
}

@end

@interface GoPaintView : NSView
@end

@implementation GoPaintView

- (BOOL)acceptsFirstResponder {
    return YES;
}

- (BOOL)becomeFirstResponder {
    return YES;
}

- (void)keyUp:(NSEvent *)event {
    NSString *chars = [event characters];
    if ([chars length] > 0) {
        unichar c = [chars characterAtIndex:0];
        go_on_key_up((int)c);
    }
}

- (void)keyDown:(NSEvent *)event {
    NSString *chars = [event characters];
    if ([chars length] > 0) {
        unichar ch = [chars characterAtIndex:0];
        go_on_char((int)ch);
    }

    go_on_key_down((int)[event keyCode]);
}

- (void)flagsChanged:(NSEvent *)event {
    NSEventModifierFlags flags = [event modifierFlags];

    int shift = (flags & NSEventModifierFlagShift) ? 1 : 0;
    int ctrl  = (flags & NSEventModifierFlagControl) ? 1 : 0;
    int alt   = (flags & NSEventModifierFlagOption) ? 1 : 0;
    int cmd   = (flags & NSEventModifierFlagCommand) ? 1 : 0;

    go_on_modifier_change(shift, ctrl, alt, cmd);
}

- (void)mouseDown:(NSEvent *)event {
    NSPoint p = [self convertPoint:[event locationInWindow] fromView:nil];
    if ([event clickCount] == 2) {
        go_on_mouse_double_click(0, (int)p.x, (int)p.y); // Left
    } else {
        go_on_mouse_down(0, (int)p.x, (int)p.y);
    }
}

- (void)rightMouseDown:(NSEvent *)event {
    NSPoint p = [self convertPoint:[event locationInWindow] fromView:nil];
    if ([event clickCount] == 2) {
        go_on_mouse_double_click(1, (int)p.x, (int)p.y); // Right
    } else {
        go_on_mouse_down(1, (int)p.x, (int)p.y);
    }
}

- (void)otherMouseDown:(NSEvent *)event {
    NSPoint p = [self convertPoint:[event locationInWindow] fromView:nil];
    if ([event clickCount] == 2) {
        go_on_mouse_double_click(2, (int)p.x, (int)p.y); // Middle/Other
    } else {
        go_on_mouse_down(2, (int)p.x, (int)p.y);
    }
}

- (void)mouseUp:(NSEvent *)event {
    NSPoint p = [self convertPoint:[event locationInWindow] fromView:nil];
    go_on_mouse_up(0, (int)p.x, (int)p.y);
}

- (void)rightMouseUp:(NSEvent *)event {
    NSPoint p = [self convertPoint:[event locationInWindow] fromView:nil];
    go_on_mouse_up(1, (int)p.x, (int)p.y);
}

- (void)otherMouseUp:(NSEvent *)event {
    NSPoint p = [self convertPoint:[event locationInWindow] fromView:nil];
    go_on_mouse_up(2, (int)p.x, (int)p.y);
}

- (void)mouseMoved:(NSEvent *)event {
    NSPoint p = [self convertPoint:[event locationInWindow] fromView:nil];
    go_on_mouse_move((int)p.x, (int)p.y);
}

- (void)mouseDragged:(NSEvent *)event {
    [self mouseMoved:event]; // тот же вызов
}

- (void)scrollWheel:(NSEvent *)event {
    float deltaY = [event deltaY];
    go_on_mouse_scroll((int)deltaY);
}

- (void)mouseEntered:(NSEvent *)event {
    go_on_mouse_enter();
}

- (void)mouseExited:(NSEvent *)event {
    go_on_mouse_leave();
}

- (void)updateTrackingAreas {
    [super updateTrackingAreas];

    NSTrackingArea *trackingArea = [[NSTrackingArea alloc] initWithRect:self.bounds
                                                                options:(NSTrackingMouseEnteredAndExited |
                                                                         NSTrackingMouseMoved |
                                                                         NSTrackingActiveAlways |
                                                                         NSTrackingInVisibleRect)
                                                                  owner:self
                                                               userInfo:nil];
    [self addTrackingArea:trackingArea];
}



- (BOOL)isFlipped { return NO; }

static void buffer_release_callback(void* info, const void* data, size_t size) {
    free((void*)data);
}

- (void)drawRect:(NSRect)dirtyRect {
    int width = (int)self.bounds.size.width;
    int height = (int)self.bounds.size.height;
    int stride = width * 4;
    size_t dataSize = stride * height;

    uint8_t* buffer = (uint8_t*)malloc(dataSize);
    if (!buffer) return;

    memset(buffer, 255, dataSize); 

    int windowId = (int)[self.window windowNumber];
    go_on_paint(buffer, width, height, windowId);

    CGContextRef ctx = [[NSGraphicsContext currentContext] CGContext];
    CGColorSpaceRef colorSpace = CGColorSpaceCreateDeviceRGB();
    CGDataProviderRef provider = CGDataProviderCreateWithData(NULL, buffer, dataSize, buffer_release_callback);

    CGImageRef image = CGImageCreate(width, height, 8, 32, stride, colorSpace,
                                     kCGImageAlphaPremultipliedLast | kCGBitmapByteOrder32Big,
                                     provider, NULL, false, kCGRenderingIntentDefault);

    CGContextSaveGState(ctx);

    CGContextDrawImage(ctx, CGRectMake(0, 0, width, height), image);

    CGContextRestoreGState(ctx);

    CGImageRelease(image);
    CGDataProviderRelease(provider);
    CGColorSpaceRelease(colorSpace);
}

@end

int InitWindow(void) {
    @autoreleasepool {
        NSApplication *app = [NSApplication sharedApplication];
        [app setActivationPolicy:NSApplicationActivationPolicyRegular];

        AppDelegate *delegate = [[AppDelegate alloc] init];
        [app setDelegate:delegate];

        NSRect frame = NSMakeRect(100, 100, 800, 600);
        NSUInteger style = NSWindowStyleMaskTitled | NSWindowStyleMaskClosable | NSWindowStyleMaskResizable;

        window = [[NSWindow alloc] initWithContentRect:frame
                                             styleMask:style
                                               backing:NSBackingStoreBuffered
                                                 defer:NO];

        GoPaintView *view = [[GoPaintView alloc] initWithFrame:frame];
        [window setContentView:view];
        [window setTitle:@"Paint from Go"];
        [window makeKeyAndOrderFront:nil];

        [app activateIgnoringOtherApps:YES];
        return (int)[window windowNumber];
    }
}

void RunEventLoop(void) {
    @autoreleasepool {
        [[NSApplication sharedApplication] run];
    }
}
