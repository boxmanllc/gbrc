#ifndef GB_JOYPAD_H
#define GB_JOYPAD_H

#include <stdbool.h>
#include <stdint.h>

#define FACE_SELECT_BIT 5
#define DPAD_SELECT_BIT 4

typedef enum {
  BUTTON_UNKNOWN = -1,
  DOWN = 7, UP = 6, LEFT = 5, RIGHT = 4,
  START = 3, SELECT = 2, B = 1, A = 0,
} Button;

extern const Button DPAD_BUTTONS[4];
extern const Button FACE_BUTTONS[4];

typedef struct {
	bool buttons[8];
	bool dpad_selected;
	bool face_selected;
} Jp;

void joypad_init_impl(Jp *pad);
void joypad_write_impl(Jp *pad, uint8_t val);
uint8_t joypad_read_impl(Jp *pad);
void joypad_press_impl(Jp *pad, Button button, bool pressed);

void joypad_init();
uint8_t joypad_read();
void joypad_write(uint8_t val);
void joypad_press(Button b, bool pressed);

#endif
