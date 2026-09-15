#ifndef GB_FRONTEND_H
#define GB_FRONTEND_H

#include <stdbool.h>
#include <stdint.h>

bool frontend_init(void);
void frontend_close(void);
void frontend_poll(void);
void frontend_present(const uint8_t *framebuffer);
bool frontend_should_quit(void);
void frontend_wait_frame(void);

#endif
