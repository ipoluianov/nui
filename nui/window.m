#import <Cocoa/Cocoa.h>
#import "window.h"

static NSWindow* window;

@interface GoPaintView : NSView
@end

@implementation GoPaintView

- (BOOL)isFlipped { return YES; }

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
    CGContextTranslateCTM(ctx, 0, height);
    CGContextScaleCTM(ctx, 1, -1);

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
