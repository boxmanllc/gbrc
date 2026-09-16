#ifndef GB_PROFILE_H
#define GB_PROFILE_H

#include <stdint.h>

void profile_record(uint16_t pc);
void profile_dump(const char *path);

#endif
