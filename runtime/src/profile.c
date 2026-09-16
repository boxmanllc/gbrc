#include "profile.h"
#include <stdio.h>

static uint8_t profile_map[0x10000];

void profile_record(uint16_t pc) { profile_map[pc] = 1; }

void profile_dump(const char *path) {
	if (!path) {
		return;
	}

	FILE *f = fopen(path, "w");
	if (!f) {
		return;
	}

	for (int a = 0; a < 0x10000; a++) {
		if (profile_map[a]) {
			fprintf(f, "%04X\n", a);
		}
	}

	fclose(f);
}
