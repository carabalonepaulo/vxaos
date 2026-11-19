#ifndef __VXAOS_INPUT_TEST_H__
#define __VXAOS_INPUT_TEST_H__
//--------------------
// Optional, test-only input injection. It is disabled unless the process is
// started with VXAOS_TEST_INPUT set to a TCP port (for example "5001"). When
// enabled, a localhost-only listener accepts one command per line and feeds
// keyboard/mouse state into Input__update/Mouse__update. Normal play is not
// affected because the listener never starts without the environment variable.
//--------------------
#include <stdint.h>

// Starts the listener. Safe to call more than once; only the first call with a
// non-empty VXAOS_TEST_INPUT has an effect.
void VxaosInputTest__start(void);

// ORs injected key-down states into the 256-byte keyboard state array.
void VxaosInputTest__apply_keyboard(uint8_t state[256]);

// Returns 1 and writes client coordinates when a mouse position override is
// active; returns 0 to use the physical cursor.
int VxaosInputTest__mouse_position(long *x, long *y);

// Ruby-facing version of the same override. pos receives {x, y} in client
// coordinates.
__declspec(dllexport) int VxaosInputTest__mousePos(long *pos);

// Returns 1 while the test bridge is active. Input__update then ignores the
// physical keyboard so automation is deterministic even if the user is typing.
int VxaosInputTest__isolateKeyboard(void);

#endif
