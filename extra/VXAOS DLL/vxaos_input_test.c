//--------------------
// Test-only input injection for VXAOS clients.
//
// The game reads keyboard state with GetKeyboardState and the cursor with
// GetCursorPos, so PostMessage/SendInput either do nothing or steal the user's
// focus. This module instead keeps a small override state that Input__update
// and Mouse__update merge with the physical devices. It is enabled only when
// VXAOS_TEST_INPUT names a TCP port, and listens on 127.0.0.1 only.
//--------------------
#include <winsock2.h>
#include <ws2tcpip.h>
#include <windows.h>
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <ctype.h>

#include "vxaos_input_test.h"

#define VIT_KEY_COUNT 256


static volatile LONG  vit_keys[VIT_KEY_COUNT];
static volatile LONG  vit_key_pulse[VIT_KEY_COUNT];
static volatile LONG  vit_key_release[VIT_KEY_COUNT];
static volatile LONG  vit_mouse_valid;
static volatile LONG  vit_mouse_until;
static volatile LONG  vit_mouse_x;
static volatile LONG  vit_mouse_y;
static volatile LONG  vit_running;
static volatile LONG  vit_started;
static int            vit_port;
static HANDLE         vit_thread;

static int vit_key_from_name(const char *name) {
  //----------
  if (!name || !*name) {
    return -1;
  }
  if (!strcmp(name, "enter"))     return VK_RETURN;
  if (!strcmp(name, "esc"))       return VK_ESCAPE;
  if (!strcmp(name, "space"))     return VK_SPACE;
  if (!strcmp(name, "tab"))       return VK_TAB;
  if (!strcmp(name, "backspace")) return VK_BACK;
  if (!strcmp(name, "shift"))     return VK_SHIFT;
  if (!strcmp(name, "ctrl"))      return VK_CONTROL;
  if (!strcmp(name, "alt"))       return VK_MENU;
  if (!strcmp(name, "left"))      return VK_LEFT;
  if (!strcmp(name, "up"))        return VK_UP;
  if (!strcmp(name, "right"))     return VK_RIGHT;
  if (!strcmp(name, "down"))      return VK_DOWN;
  if (!strcmp(name, "lbutton"))   return VK_LBUTTON;
  if (!strcmp(name, "rbutton"))   return VK_RBUTTON;
  if (!strcmp(name, "mbutton"))   return VK_MBUTTON;
  //-----
  // Single letters and digits use their Windows virtual-key code (A-Z, 0-9).
  if (name[1] == '\0' && isalnum((unsigned char)name[0])) {
    return toupper((unsigned char)name[0]);
  }
  //-----
  char *end = NULL;
  long value;
  if (name[0] == '0' && (name[1] == 'x' || name[1] == 'X')) {
    value = strtol(name + 2, &end, 16);
  } else {
    value = strtol(name, &end, 10);
  }
  if (!end || *end != '\0' || value < 0 || value >= VIT_KEY_COUNT) {
    return -1;
  }
  return (int)value;
  //----------
}

static void vit_key_set(int vk, int down, int hold_ms) {
  //----------
  if (vk < 0 || vk >= VIT_KEY_COUNT) {
    return;
  }
  InterlockedExchange(&vit_keys[vk], down ? 1 : 0);
  InterlockedExchange(&vit_key_release[vk],
    (down && hold_ms > 0) ? (LONG)(GetTickCount() + (DWORD)hold_ms) : 0);
  //----------
}

// A pulse is visible for exactly one Input__update frame. It is used for text
// and clicks so the engine's auto-repeat threshold can never turn one injected
// key into several characters, regardless of the frame rate.
static void vit_pulse_key(int vk) {
  //----------
  if (vk < 0 || vk >= VIT_KEY_COUNT) {
    return;
  }
  InterlockedExchange(&vit_keys[vk], 0);
  InterlockedExchange(&vit_key_release[vk], 0);
  InterlockedExchange(&vit_key_pulse[vk], 1);
  //----------
}

static void vit_press(int vk, int hold_ms) {
  //----------
  if (hold_ms <= 0) {
    hold_ms = 50;
  }
  vit_key_set(vk, 1, 0);
  Sleep(hold_ms);
  vit_key_set(vk, 0, 0);
  Sleep(25);
  //----------
}

static void vit_type_char(unsigned char c) {
  //----------
  SHORT scan = VkKeyScanA(c);
  if (scan == -1) {
    return;
  }
  int vk = scan & 0xFF;
  int shift = (scan >> 8) & 1;
  if (shift) {
    vit_key_set(VK_SHIFT, 1, 0);
    Sleep(5);
  }
  vit_pulse_key(vk);
  Sleep(35);
  if (shift) {
    vit_key_set(VK_SHIFT, 0, 0);
  }
  Sleep(45);
  //----------
}

static void vit_click(int x, int y, const char *button, int hold_ms) {
  //----------
  InterlockedExchange(&vit_mouse_x, x);
  InterlockedExchange(&vit_mouse_y, y);
  InterlockedExchange(&vit_mouse_valid, 1);
  InterlockedExchange(&vit_mouse_until, (LONG)(GetTickCount() + 2000));
  Sleep(30);
  //-----
  int vk = VK_LBUTTON;
  if (button && (button[0] == 'r' || button[0] == 'R')) vk = VK_RBUTTON;
  else if (button && (button[0] == 'm' || button[0] == 'M')) vk = VK_MBUTTON;
  vit_pulse_key(vk);
  Sleep(hold_ms > 0 ? hold_ms : 40);
  //----------
}

static void vit_reset(void) {
  //----------
  for (int i = 0; i < VIT_KEY_COUNT; i++) {
    vit_key_set(i, 0, 0);
    InterlockedExchange(&vit_key_pulse[i], 0);
  }
  InterlockedExchange(&vit_mouse_valid, 0);
  //----------
}

static void vit_handle_line(char *line) {
  //----------
  char *cmd = strtok(line, " \t\r\n");
  if (!cmd) {
    return;
  }
  if (!strcmp(cmd, "key")) {
    char *name = strtok(NULL, " \t\r\n");
    char *action = strtok(NULL, " \t\r\n");
    char *hold = strtok(NULL, " \t\r\n");
    int vk = vit_key_from_name(name);
    if (vk < 0 || !action) {
      return;
    }
    if (!strcmp(action, "down")) {
      vit_key_set(vk, 1, 0);
    } else if (!strcmp(action, "up")) {
      vit_key_set(vk, 0, 0);
    } else {
      vit_press(vk, hold ? atoi(hold) : 50);
    }
  } else if (!strcmp(cmd, "type")) {
    char *text = strtok(NULL, "");
    if (text) {
      while (*text == ' ' || *text == '\t') {
        text++;
      }
      for (; *text; text++) {
        if (*text == '\r' || *text == '\n') {
          vit_press(VK_RETURN, 60);
        } else if (*text >= 32 && *text < 127) {
          vit_type_char((unsigned char)*text);
        }
      }
    }
  } else if (!strcmp(cmd, "move")) {
    char *xs = strtok(NULL, " \t\r\n");
    char *ys = strtok(NULL, " \t\r\n");
    if (xs && ys) {
      InterlockedExchange(&vit_mouse_x, atoi(xs));
      InterlockedExchange(&vit_mouse_y, atoi(ys));
      InterlockedExchange(&vit_mouse_valid, 1);
      InterlockedExchange(&vit_mouse_until, (LONG)(GetTickCount() + 2000));
    }
  } else if (!strcmp(cmd, "click")) {
    char *xs = strtok(NULL, " \t\r\n");
    char *ys = strtok(NULL, " \t\r\n");
    char *button = strtok(NULL, " \t\r\n");
    char *hold = strtok(NULL, " \t\r\n");
    if (xs && ys) {
      vit_click(atoi(xs), atoi(ys), button, hold ? atoi(hold) : 80);
    }
  } else if (!strcmp(cmd, "reset")) {
    vit_reset();
  }
  //----------
}

static DWORD WINAPI vit_server(LPVOID unused) {
  //----------
  (void)unused;
  WSADATA wsa;
  if (WSAStartup(MAKEWORD(2, 2), &wsa) != 0) {
    return 0;
  }
  //-----
  SOCKET listener = socket(AF_INET, SOCK_STREAM, IPPROTO_TCP);
  if (listener == INVALID_SOCKET) {
    WSACleanup();
    return 0;
  }
  BOOL reuse = TRUE;
  setsockopt(listener, SOL_SOCKET, SO_REUSEADDR, (const char *)&reuse, sizeof(reuse));
  //-----
  struct sockaddr_in addr;
  memset(&addr, 0, sizeof(addr));
  addr.sin_family = AF_INET;
  addr.sin_port = htons((u_short)vit_port);
  addr.sin_addr.s_addr = htonl(INADDR_LOOPBACK);
  if (bind(listener, (struct sockaddr *)&addr, sizeof(addr)) != 0 ||
      listen(listener, 4) != 0) {
    closesocket(listener);
    WSACleanup();
    return 0;
  }
  //-----
  while (vit_running) {
    SOCKET client = accept(listener, NULL, NULL);
    if (client == INVALID_SOCKET) {
      if (!vit_running) {
        break;
      }
      Sleep(100);
      continue;
    }
    char chunk[512];
    char line[2048];
    size_t used = 0;
    for (;;) {
      int got = recv(client, chunk, sizeof(chunk), 0);
      if (got <= 0) {
        break;
      }
      for (int i = 0; i < got; i++) {
        char ch = chunk[i];
        if (ch == '\n') {
          line[used] = '\0';
          vit_handle_line(line);
          used = 0;
        } else if (used < sizeof(line) - 1) {
          line[used++] = ch;
        }
      }
    }
    if (used > 0) {
      line[used] = '\0';
      vit_handle_line(line);
    }
    closesocket(client);
  }
  //-----
  closesocket(listener);
  WSACleanup();
  return 0;
  //----------
}

void VxaosInputTest__start(void) {
  //----------
  if (InterlockedCompareExchange(&vit_started, 1, 0) != 0) {
    return;
  }
  const char *value = getenv("VXAOS_TEST_INPUT");
  if (!value || !*value) {
    return;
  }
  // Accept "5001" or "127.0.0.1:5001".
  const char *colon = strrchr(value, ':');
  vit_port = atoi(colon ? colon + 1 : value);
  if (vit_port <= 0 || vit_port > 65535) {
    return;
  }
  InterlockedExchange(&vit_running, 1);
  vit_thread = CreateThread(NULL, 0, vit_server, NULL, 0, NULL);
  //----------
}

void VxaosInputTest__apply_keyboard(uint8_t state[256]) {
  //----------
  if (!vit_running) {
    return;
  }
  DWORD now = GetTickCount();
  for (int i = 0; i < VIT_KEY_COUNT; i++) {
    DWORD release = vit_key_release[i];
    if (release != 0 && (LONG)(now - release) >= 0) {
      InterlockedExchange(&vit_key_release[i], 0);
      InterlockedExchange(&vit_keys[i], 0);
    }
    if (vit_key_pulse[i]) {
      state[i] |= 0x80;
      InterlockedExchange(&vit_key_pulse[i], 0);
    } else if (vit_keys[i]) {
      state[i] |= 0x80;
    }
  }
  //----------
}

int VxaosInputTest__mouse_position(long *x, long *y) {
  //----------
  if (!vit_running || !vit_mouse_valid) {
    return 0;
  }
  if ((LONG)(GetTickCount() - (DWORD)vit_mouse_until) >= 0) {
    InterlockedExchange(&vit_mouse_valid, 0);
    return 0;
  }
  *x = vit_mouse_x;
  *y = vit_mouse_y;
  return 1;
  //----------
}

__declspec(dllexport) int VxaosInputTest__mousePos(long *pos) {
  //----------
  long ix = 0, iy = 0;
  if (!VxaosInputTest__mouse_position(&ix, &iy)) {
    return 0;
  }
  if (pos) {
    pos[0] = ix;
    pos[1] = iy;
  }
  return 1;
  //----------
}

int VxaosInputTest__isolateKeyboard(void) {
  return vit_running != 0;
}
