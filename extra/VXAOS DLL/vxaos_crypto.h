#ifndef __VXAOS_CRYPTO_H__
#define __VXAOS_CRYPTO_H__

#include "main.h"

#ifdef __cplusplus
extern "C" {
#endif

void DLL_EXPORT Crypto__sha256(const unsigned char *data, int len, unsigned char *out);
void DLL_EXPORT Crypto__hmac_sha256(const unsigned char *key, int key_len,
                                    const unsigned char *data, int data_len,
                                    unsigned char *out);
void DLL_EXPORT Crypto__pbkdf2_sha256(const unsigned char *password, int pass_len,
                                      const unsigned char *salt, int salt_len,
                                      int iterations, unsigned char *out, int out_len);

#ifdef __cplusplus
}
#endif

#endif
