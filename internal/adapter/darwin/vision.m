// vision.m — native on-screen OCR via the macOS Vision framework.
//
// Compiled by cgo as Objective-C (LDFLAGS add -framework Vision in the
// Go file that includes vision.h). Captures a global point rect with
// CGWindowListCreateImage (same path as the PNG capture), then runs a
// synchronous VNRecognizeTextRequest over the CGImage.
//
// Coordinate transform (the crux): Vision returns each observation's
// boundingBox as a normalized rect in [0,1] with a BOTTOM-LEFT origin.
// We flip it to a top-left image space and scale by the POINT dimensions
// (ow,oh) — not the pixel dimensions — because the whole visual loop is
// point-normalized (1 image pixel == 1 global point). The centroid is
// returned relative to the capture origin; Go adds (ox,oy).
#import <Vision/Vision.h>
#import <CoreGraphics/CoreGraphics.h>
#import <ApplicationServices/ApplicationServices.h>
#include "vision.h"
#include <string.h>

#pragma clang diagnostic ignored "-Wdeprecated-declarations"

int wctl_ocr_rect(double ox, double oy, double ow, double oh,
                  wctl_ocr_match *out, int maxout) {
    if (!CGPreflightScreenCaptureAccess()) return -1;
    if (ow <= 0 || oh <= 0 || maxout <= 0) return -5;

  int n = 0;
  @autoreleasepool {
    CGImageRef img = CGWindowListCreateImage(
        CGRectMake(ox, oy, ow, oh),
        kCGWindowListOptionOnScreenOnly, kCGNullWindowID, kCGWindowImageDefault);
    if (!img) return -5;

    VNRecognizeTextRequest *req = [[VNRecognizeTextRequest alloc] init];
    req.recognitionLevel = VNRequestTextRecognitionLevelAccurate;
    req.usesLanguageCorrection = NO;

    VNImageRequestHandler *handler =
        [[VNImageRequestHandler alloc] initWithCGImage:img options:@{}];
    NSError *err = nil;
    BOOL ok = [handler performRequests:@[req] error:&err];
    CGImageRelease(img);
    if (!ok) { [req release]; [handler release]; return -5; }

    NSArray<VNRecognizedTextObservation *> *obs = req.results;
    for (VNRecognizedTextObservation *o in obs) {
        if (n >= maxout) break;
        VNRecognizedText *cand = [[o topCandidates:1] firstObject];
        if (!cand) continue;
        NSString *s = cand.string;
        if (s.length == 0) continue;

        // boundingBox: normalized [0,1], bottom-left origin.
        CGRect bb = o.boundingBox;
        double bx = bb.origin.x;
        double bw = bb.size.width;
        double bh = bb.size.height;
        // Flip Y to top-left: image-top y = (1 - (by + bh)) = 1 - bb.maxY.
        double topY = 1.0 - (bb.origin.y + bh);

        wctl_ocr_match *m = &out[n];
        m->x = bx * ow;
        m->y = topY * oh;
        m->w = bw * ow;
        m->h = bh * oh;
        m->cx = m->x + m->w / 2.0;
        m->cy = m->y + m->h / 2.0;
        m->confidence = cand.confidence;

        const char *utf8 = [s UTF8String];
        if (utf8) {
            strncpy(m->text, utf8, sizeof(m->text) - 1);
            m->text[sizeof(m->text) - 1] = '\0';
        } else {
            m->text[0] = '\0';
        }
        n++;
    }
    [req release];
    [handler release];
  }
    return n;
}
