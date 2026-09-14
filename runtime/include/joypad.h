#ifndef GB_JOYPAD_H
#define GB_JOYPAD_H

#include <stdbool.h>
#include <stdint.h>

#define IO_START 0xFF00
#define IO_END 0xFF3F
#define IO_SIZE (IO_END - IO_START + 1)

#define JOYPAD_ADDR 0xFF00
#define FACE_SELECT_BIT 5
#define DPAD_SELECT_BIT 4

// clang-format off
typedef enum {
  DOWN = 7, UP = 6, LEFT = 5, RIGHT = 4,
  START = 3, SELECT = 2, B = 1, A = 0,
} Button;
// clang-format on

extern const Button DPAD_BUTTONS[4];
extern const Button FACE_BUTTONS[4];

typedef struct {
	bool buttons[8];
	bool dpad_selected;
	bool face_selected;
	uint8_t ram[IO_SIZE];
} Jp;

uint8_t read_u8(Jp *pad, uint16_t addr);
void joypad_init_impl(Jp *pad);
void joypad_write_impl(Jp *pad, uint16_t addr, uint8_t val);
uint8_t joypad_read_impl(Jp *pad);
void joypad_press_impl(Jp *pad, Button button, bool pressed);

// no arg wrappers
void joypad_init();
uint8_t joypad_read();
void joypad_write(uint8_t val);
void joypad_press(Button b, bool pressed);

#endif
