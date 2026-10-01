#import <EventKit/EventKit.h>
#import <CoreGraphics/CoreGraphics.h>
#include <stdlib.h>
#include <string.h>
#include "bridge.h"

extern void sundialStoreChanged(void);

static EKEventStore *sdStore(void) {
    static EKEventStore *store;
    static dispatch_once_t once;
    dispatch_once(&once, ^{ store = [[EKEventStore alloc] init]; });
    return store;
}

int sd_auth_status(void) {
    return (int)[EKEventStore authorizationStatusForEntityType:EKEntityTypeEvent];
}

int sd_request_access(void) {
    __block BOOL granted = NO;
    dispatch_semaphore_t sem = dispatch_semaphore_create(0);
    [sdStore() requestFullAccessToEventsWithCompletion:^(BOOL ok, NSError *err) {
        granted = ok;
        dispatch_semaphore_signal(sem);
    }];
    dispatch_semaphore_wait(sem, DISPATCH_TIME_FOREVER);
    return granted ? 1 : 0;
}

static NSString *sdHex(CGColorRef color) {
    if (color == NULL) return @"";
    CGColorSpaceRef srgb = CGColorSpaceCreateWithName(kCGColorSpaceSRGB);
    CGColorRef conv = CGColorCreateCopyByMatchingToColorSpace(srgb, kCGRenderingIntentDefault, color, NULL);
    CGColorSpaceRelease(srgb);
    if (conv == NULL) return @"";
    NSString *hex = @"";
    if (CGColorGetNumberOfComponents(conv) >= 3) {
        const CGFloat *c = CGColorGetComponents(conv);
        hex = [NSString stringWithFormat:@"#%02X%02X%02X",
               (int)lround(c[0] * 255), (int)lround(c[1] * 255), (int)lround(c[2] * 255)];
    }
    CGColorRelease(conv);
    return hex;
}

static char *sdJSON(id obj) {
    NSData *data = [NSJSONSerialization dataWithJSONObject:obj options:0 error:nil];
    if (data == nil) return NULL;
    char *out = malloc(data.length + 1);
    memcpy(out, data.bytes, data.length);
    out[data.length] = 0;
    return out;
}

static NSString *sdDay(NSCalendar *cal, NSDate *d) {
    NSDateComponents *c = [cal components:NSCalendarUnitYear | NSCalendarUnitMonth | NSCalendarUnitDay fromDate:d];
    return [NSString stringWithFormat:@"%04ld-%02ld-%02ld", (long)c.year, (long)c.month, (long)c.day];
}

char *sd_calendars(void) {
    @autoreleasepool {
        NSMutableArray *out = [NSMutableArray array];
        for (EKCalendar *cal in [sdStore() calendarsForEntityType:EKEntityTypeEvent]) {
            [out addObject:@{
                @"id": cal.calendarIdentifier ?: @"",
                @"title": cal.title ?: @"",
                @"color": sdHex(cal.CGColor),
                @"source": cal.source.title ?: @"",
                @"readOnly": @((BOOL)!cal.allowsContentModifications),
            }];
        }
        return sdJSON(out);
    }
}

char *sd_events(double start, double end) {
    @autoreleasepool {
        NSDate *s = [NSDate dateWithTimeIntervalSince1970:start];
        NSDate *e = [NSDate dateWithTimeIntervalSince1970:end];
        NSPredicate *p = [sdStore() predicateForEventsWithStartDate:s endDate:e calendars:nil];
        NSCalendar *cal = [NSCalendar currentCalendar];
        NSMutableArray *out = [NSMutableArray array];
        for (EKEvent *ev in [sdStore() eventsMatchingPredicate:p]) {
            NSMutableArray *people = [NSMutableArray array];
            for (EKParticipant *pt in ev.attendees) {
                if (pt.name.length > 0) [people addObject:pt.name];
            }
            NSMutableDictionary *d = [@{
                @"id": ev.eventIdentifier ?: @"",
                @"calendarId": ev.calendar.calendarIdentifier ?: @"",
                @"title": ev.title ?: @"",
                @"start": @(ev.startDate.timeIntervalSince1970),
                @"end": @(ev.endDate.timeIntervalSince1970),
                @"allDay": @(ev.allDay),
                @"location": ev.location ?: @"",
                @"notes": ev.notes ?: @"",
                @"url": ev.URL.absoluteString ?: @"",
                @"attendees": people,
                @"status": @((int)ev.status),
            } mutableCopy];
            if (ev.allDay) {
                // EventKit reports all-day ends as 23:59:59 of the last day or
                // midnight after it. Normalize to an exclusive end date.
                NSDate *lastInstant = [ev.endDate dateByAddingTimeInterval:-1];
                if ([lastInstant compare:ev.startDate] == NSOrderedAscending) lastInstant = ev.startDate;
                NSDate *lastDay = [cal startOfDayForDate:lastInstant];
                NSDate *endExclusive = [cal dateByAddingUnit:NSCalendarUnitDay value:1 toDate:lastDay options:0];
                d[@"startDay"] = sdDay(cal, ev.startDate);
                d[@"endDay"] = sdDay(cal, endExclusive);
            }
            [out addObject:d];
        }
        return sdJSON(out);
    }
}

static id sdObserver;

void sd_watch_start(void) {
    if (sdObserver != nil) return;
    NSOperationQueue *q = [[NSOperationQueue alloc] init];
    sdObserver = [[NSNotificationCenter defaultCenter]
        addObserverForName:EKEventStoreChangedNotification
                    object:sdStore()
                     queue:q
                usingBlock:^(NSNotification *n) { sundialStoreChanged(); }];
}

void sd_watch_stop(void) {
    if (sdObserver == nil) return;
    [[NSNotificationCenter defaultCenter] removeObserver:sdObserver];
    sdObserver = nil;
}

void sd_free(char *p) { free(p); }
