//go:build darwin
// +build darwin

#import <Cocoa/Cocoa.h>
#import <mach/mach_time.h>
#import "window.h"
#include <pthread.h>

// static NSWindow* window;

static NSMutableDictionary<NSNumber*, NSWindow*> *windowMap;
static NSMutableDictionary<NSNumber*, NSTimer*> *timers;

void Log(int code) {
}

__attribute__((constructor))
static void InitWindowMap() {
    windowMap = [NSMutableDictionary new];
}

@interface AppDelegate : NSObject <NSApplicationDelegate>
@end

@implementation AppDelegate

- (BOOL)applicationShouldTerminateAfterLastWindowClosed:(NSApplication *)sender {
    return YES;
}

- (void)windowDidMove:(NSNotification *)notification {
    NSWindow *window = notification.object;
    int windowId = (int)window.windowNumber;
    NSPoint pos = [window frame].origin;
    go_on_window_move(windowId, (int)pos.x, (int)pos.y); // вызов в Go
}


@end

@interface GoPaintView : NSView
@property (strong) NSTrackingArea *trackingArea;
@end

@implementation GoPaintView

- (BOOL)acceptsFirstResponder {
    return YES;
}

- (BOOL)becomeFirstResponder {
    return YES;
}

- (void)setFrameSize:(NSSize)newSize {
    [super setFrameSize:newSize];

    int windowId = (int)[self.window windowNumber];
    go_on_resize(windowId, (int)newSize.width, (int)newSize.height);
}

- (void)keyUp:(NSEvent *)event {
    NSString *chars = [event characters];
    if ([chars length] > 0) {
        //unichar c = [chars characterAtIndex:0];
        go_on_key_up((int)[self.window windowNumber], (int)[event keyCode]);
    }
}

- (void)keyDown:(NSEvent *)event {
    NSString *chars = [event characters];
    if ([chars length] > 0) {
        unichar ch = [chars characterAtIndex:0];
        go_on_char((int)[self.window windowNumber], (int)ch);
    }

    go_on_key_down((int)[self.window windowNumber], (int)[event keyCode]);
}

- (void)flagsChanged:(NSEvent *)event {
    NSEventModifierFlags flags = [event modifierFlags];

    int shift = (flags & NSEventModifierFlagShift) ? 1 : 0;
    int ctrl  = (flags & NSEventModifierFlagControl) ? 1 : 0;
    int alt   = (flags & NSEventModifierFlagOption) ? 1 : 0;
    int cmd   = (flags & NSEventModifierFlagCommand) ? 1 : 0;
    int capsLock = (flags & NSEventModifierFlagCapsLock) ? 1 : 0;
    int numPad = (flags & NSEventModifierFlagNumericPad) ? 1 : 0;
    int fnKey = (flags & NSEventModifierFlagFunction) ? 1 : 0;

    go_on_modifier_change((int)[self.window windowNumber], shift, ctrl, alt, cmd, capsLock, numPad, fnKey);
}

- (void)mouseDown:(NSEvent *)event {
    NSPoint p = [self convertPoint:[event locationInWindow] fromView:nil];
    if ([event clickCount] == 2) {
        go_on_mouse_double_click((int)[self.window windowNumber], 0, (int)p.x, (int)p.y); // Left
    } else {
        go_on_mouse_down((int)[self.window windowNumber], 0, (int)p.x, (int)p.y);
    }
}

- (void)rightMouseDown:(NSEvent *)event {
    NSPoint p = [self convertPoint:[event locationInWindow] fromView:nil];
    if ([event clickCount] == 2) {
        go_on_mouse_double_click((int)[self.window windowNumber], 1, (int)p.x, (int)p.y); // Right
    } else {
        go_on_mouse_down((int)[self.window windowNumber], 1, (int)p.x, (int)p.y);
    }
}

- (void)otherMouseDown:(NSEvent *)event {
    NSPoint p = [self convertPoint:[event locationInWindow] fromView:nil];
    if ([event clickCount] == 2) {
        go_on_mouse_double_click((int)[self.window windowNumber], 2, (int)p.x, (int)p.y); // Middle/Other
    } else {
        go_on_mouse_down((int)[self.window windowNumber], 2, (int)p.x, (int)p.y);
    }
}

- (void)mouseUp:(NSEvent *)event {
    NSPoint p = [self convertPoint:[event locationInWindow] fromView:nil];
    go_on_mouse_up((int)[self.window windowNumber], 0, (int)p.x, (int)p.y);
}

- (void)rightMouseUp:(NSEvent *)event {
    NSPoint p = [self convertPoint:[event locationInWindow] fromView:nil];
    go_on_mouse_up((int)[self.window windowNumber], 1, (int)p.x, (int)p.y);
}

- (void)otherMouseUp:(NSEvent *)event {
    NSPoint p = [self convertPoint:[event locationInWindow] fromView:nil];
    go_on_mouse_up((int)[self.window windowNumber], 2, (int)p.x, (int)p.y);
}

- (void)mouseMoved:(NSEvent *)event {
    NSPoint p = [self convertPoint:[event locationInWindow] fromView:nil];
    go_on_mouse_move((int)[self.window windowNumber], (int)p.x, (int)p.y);
}

- (void)mouseDragged:(NSEvent *)event {
    [self mouseMoved:event];
}

- (void)rightMouseDragged:(NSEvent *)event {
    NSPoint p = [self convertPoint:[event locationInWindow] fromView:nil];
    go_on_mouse_move((int)[self.window windowNumber], (int)p.x, (int)p.y);
}

- (void)scrollWheel:(NSEvent *)event {
    float deltaX = [event deltaX];
    float deltaY = [event deltaY];
    if (deltaX == 0 && deltaY == 0) return;
    go_on_mouse_scroll((int)[self.window windowNumber], deltaX, deltaY);
}

- (void)mouseEntered:(NSEvent *)event {
    go_on_mouse_enter((int)[self.window windowNumber]);
}

- (void)mouseExited:(NSEvent *)event {
    fprintf(stderr, "Exited: \n");
    go_on_mouse_leave((int)[self.window windowNumber]);
}

- (void)updateTrackingAreas {
    [super updateTrackingAreas];

    if (self.trackingArea) {
        [self removeTrackingArea:self.trackingArea];
    }
// NSTrackingArea *
    self.trackingArea = [[NSTrackingArea alloc] initWithRect:self.bounds
                                                                options:(NSTrackingMouseEnteredAndExited |
                                                                         NSTrackingMouseMoved |
                                                                         NSTrackingActiveAlways |
                                                                         NSTrackingInVisibleRect)
                                                                  owner:self
                                                               userInfo:nil];
    [self addTrackingArea:self.trackingArea];
}

- (BOOL)isFlipped { return NO; }

static void buffer_release_callback(void* info, const void* data, size_t size) {
    free((void*)data);
}

- (void)drawRect:(NSRect)dirtyRect {
    uint64_t start = mach_absolute_time();

    int width = (int)self.bounds.size.width;
    int height = (int)self.bounds.size.height;
    int stride = width * 4;
    size_t dataSize = stride * height;

    uint8_t* buffer = (uint8_t*)malloc(dataSize);
    if (!buffer) return;


    //memset(buffer, 255, dataSize); 

    go_on_paint((int)[self.window windowNumber], buffer, width, height);

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

    uint64_t end = mach_absolute_time();
    static mach_timebase_info_data_t info = {0};
    if (info.denom == 0) {
        mach_timebase_info(&info);
    }
    uint64_t elapsedNano = (end - start) * info.numer / info.denom;
    uint64_t elapsedMicro = elapsedNano / 1000;
    go_on_declare_draw_time((int)[self.window windowNumber], (int)elapsedMicro);
}

@end

int InitWindow(void) {
    @autoreleasepool {
        if (!timers) {
            timers = [[NSMutableDictionary alloc] init];
        }

        // Log to console
        Log(1);

        NSWindow* window;
        NSApplication *app = [NSApplication sharedApplication];
        [app setActivationPolicy:NSApplicationActivationPolicyRegular];

        Log(2);

        AppDelegate *delegate = [[AppDelegate alloc] init];
        [app setDelegate:delegate];

        Log(3);

        NSRect frame = NSMakeRect(100, 100, 800, 600);
        NSUInteger style = NSWindowStyleMaskTitled | NSWindowStyleMaskClosable | NSWindowStyleMaskResizable | NSWindowStyleMaskMiniaturizable;

        Log(4);

        fprintf(stderr, "Thread: %ld\n", pthread_self());
        @try {

        window = [[NSWindow alloc] initWithContentRect:frame
                                             styleMask:style
                                               backing:NSBackingStoreBuffered
                                                 defer:NO];

        } @catch (NSException *exception) {
            fprintf(stderr, "Exception: %s\n", [[exception reason] UTF8String]);
            return -1;
        }

        Log(5);


        GoPaintView *view = [[GoPaintView alloc] initWithFrame:frame];
        [window setContentView:view];
        [window setTitle:@"NUI Window"];
        [window setDelegate:delegate];

        NSLog(@"NUI: InitWindow 1");

        //[window makeKeyAndOrderFront:nil];

        [app activateIgnoringOtherApps:YES];
        int windowId = (int)[window windowNumber];
        windowMap[@(windowId)] = window;

        NSLog(@"NUI: InitWindow 1");

        return windowId;
    }
}

void RunEventLoop(void) {
    @autoreleasepool {
        [[NSApplication sharedApplication] run];
    }
}

void CloseWindowById(int windowId) {
    NSWindow *w = windowMap[@(windowId)];
    if (w) {
        [w performClose:nil];
        [windowMap removeObjectForKey:@(windowId)];
    }
}

void SetWindowTitle(int windowId, const char* title) {
    NSWindow *w = windowMap[@(windowId)];
    if (w && title) {
        NSString *nsTitle = [NSString stringWithUTF8String:title];
        [w setTitle:nsTitle];
    }
}

void SetWindowSize(int windowId, int width, int height) {
    NSWindow *w = windowMap[@(windowId)];
    if (w) {
        NSRect frame = [w frame];
        NSRect newFrame = NSMakeRect(
            frame.origin.x,
            frame.origin.y + frame.size.height - height,
            width,
            height
        );
        [w setFrame:newFrame display:YES animate:NO];
    }
}

void SetWindowPosition(int windowId, int x, int y) {
    NSWindow *w = windowMap[@(windowId)];
    if (w) {
        NSRect frame = [w frame];
        CGFloat newY = y;

        NSRect screenFrame = [[w screen] frame];
        newY = screenFrame.size.height - y - frame.size.height;

        NSRect newFrame = NSMakeRect(
            x,
            newY,
            frame.size.width,
            frame.size.height
        );

        [w setFrame:newFrame display:YES animate:NO];
    }
}

void MinimizeWindow(int windowId) {
    NSWindow *w = windowMap[@(windowId)];
    if (w) {
        [NSApp activateIgnoringOtherApps:YES];
        [w makeKeyAndOrderFront:nil];
        [w miniaturize:nil];
    }
}

void MaximizeWindow(int windowId) {
    NSWindow *w = windowMap[@(windowId)];
    if (w && ![w isZoomed]) {
        [w zoom:nil];
    }
}

void ShowWindow(int windowId) {
    NSWindow *w = windowMap[@(windowId)];
    if (!w) return;

    [w makeKeyAndOrderFront:nil];
    [NSApp activateIgnoringOtherApps:YES];
}

void SetAppIconFromRGBA(const char* data, int width, int height) {
    if (!data || width <= 0 || height <= 0) return;

    @autoreleasepool {
        NSBitmapImageRep *bitmapRep = [[NSBitmapImageRep alloc]
            initWithBitmapDataPlanes:NULL
                          pixelsWide:width
                          pixelsHigh:height
                       bitsPerSample:8
                     samplesPerPixel:4
                            hasAlpha:YES
                            isPlanar:NO
                      colorSpaceName:NSCalibratedRGBColorSpace
                         bytesPerRow:width * 4
                        bitsPerPixel:32];

        if (!bitmapRep) return;

        memcpy([bitmapRep bitmapData], data, width * height * 4);

        NSImage *image = [[NSImage alloc] initWithSize:NSMakeSize(width, height)];
        [image addRepresentation:bitmapRep];

        [NSApp setApplicationIconImage:image];
    }
}

int GetWindowPositionX(int windowId) {
    NSWindow *win = windowMap[@(windowId)];
    if (!win) return -1;
    return (int)win.frame.origin.x;
}

int GetWindowPositionY(int windowId) {
    NSWindow *win = windowMap[@(windowId)];
    if (!win) return -1;
    return (int)win.frame.origin.y;
}

int GetWindowWidth(int windowId) {
    NSWindow *win = windowMap[@(windowId)];
    if (!win) return -1;
    return (int)win.frame.size.width;
}

int GetWindowHeight(int windowId) {
    NSWindow *win = windowMap[@(windowId)];
    if (!win) return -1;
    return (int)win.frame.size.height;
}

void timerCallback(NSTimer *timer) {
    NSNumber *key = timer.userInfo;
    if (key) {
        int windowId = key.intValue;
        go_on_timer(windowId); // вызываем Go-функцию
    }
}

void StartTimer(int windowId, double intervalMilliseconds) {
    if (!timers) timers = [[NSMutableDictionary alloc] init];

    NSNumber *key = @(windowId);

    NSTimer *existing = timers[key];
    if (existing) {
        [existing invalidate];
    }

    NSTimer *timer = [NSTimer scheduledTimerWithTimeInterval:(intervalMilliseconds / 1000.0)
                                                      repeats:YES
                                                        block:^(NSTimer * _Nonnull t) {
        go_on_timer(windowId);
    }];

    [[NSRunLoop mainRunLoop] addTimer:timer forMode:NSRunLoopCommonModes];
    timers[key] = timer;
}

void StopTimer(int windowId) {
    NSNumber *key = @(windowId);
    NSTimer *timer = timers[key];
    if (timer) {
        [timer invalidate];
        [timers removeObjectForKey:key];
    }
}

void UpdateWindow(int windowId) {
    NSNumber *key = @(windowId);
    NSWindow *win = windowMap[key];
    if (win) {
        NSView *view = [win contentView];
        [view setNeedsDisplay:YES];
    }
}


int GetScreenWidth() {
    return (int)[[NSScreen mainScreen] frame].size.width;
}

int GetScreenHeight() {
    return (int)[[NSScreen mainScreen] frame].size.height;
}

void SetMacCursor(int cursorType) {
    switch (cursorType) {
        case 1: // Arrow
            [[NSCursor arrowCursor] set];
            break;
        case 2: // Pointer (hand)
            [[NSCursor pointingHandCursor] set];
            break;
        case 3: // Resize horizontal
            [[NSCursor resizeLeftRightCursor] set];
            break;
        case 4: // Resize vertical
            [[NSCursor resizeUpDownCursor] set];
            break;
        case 5: // I-beam
            [[NSCursor IBeamCursor] set];
            break;
        default:
            // Можно сбросить в Arrow или оставить как есть
            [[NSCursor arrowCursor] set];
            break;
    }
}
