// vision.h — C ABI for the native Vision-framework OCR helper.
//
// Implemented in vision.m (compiled by cgo as Objective-C). Declared in
// a plain-C header so automation.go's C preamble can call it without
// pulling Objective-C into that translation unit. See docs/specs and
// the visual-loop-hardening-and-verbs OpenSpec change.
#ifndef WCTL_VISION_H
#define WCTL_VISION_H

// One recognized-text match. All geometry is in POINTS relative to the
// capture origin (the caller adds the capture's global origin to get a
// global click point — preserving the 1 image pixel == 1 point contract).
typedef struct {
    double cx, cy;        // centroid (click point), points from capture origin
    double x, y, w, h;    // bounds, points from capture origin (top-left origin)
    double confidence;    // 0..1 from Vision's top candidate
    char   text[256];     // UTF-8, truncated
} wctl_ocr_match;

// Capture the global point rect (ox,oy,ow,oh) and run Vision OCR on it.
// Fills up to maxout matches into out (each recognized line = one match),
// returns the count. Negative on error: -1 screen-capture denied,
// -5 internal failure.
int wctl_ocr_rect(double ox, double oy, double ow, double oh,
                  wctl_ocr_match *out, int maxout);

#endif
