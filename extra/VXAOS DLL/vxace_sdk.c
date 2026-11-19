//--------------------
#include "main.h"
#include "vxace_sdk.h"
//---------------
HMODULE RGSSDLL;
pRGSSEval RGSSEval;
//---------------
void VXAceSDK__Setup(void) {
  //----------
  RGSSDLL  = LoadLibrary("./System/RGSS300.dll");
  RGSSEval = (pRGSSEval)GetProcAddress(RGSSDLL, "RGSSEval");
  //----------
}
//--------------------


