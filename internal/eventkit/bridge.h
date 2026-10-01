#ifndef SUNDIAL_BRIDGE_H
#define SUNDIAL_BRIDGE_H

int sd_auth_status(void);
int sd_request_access(void);
char *sd_calendars(void);
char *sd_events(double start, double end);
void sd_watch_start(void);
void sd_watch_stop(void);
void sd_free(char *p);

#endif
