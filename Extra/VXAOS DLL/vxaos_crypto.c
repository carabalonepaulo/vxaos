// SHA-256 / HMAC-SHA256 / PBKDF2-HMAC-SHA256 for the VXAOS client.
// Self-contained (FIPS 180-4 / RFC 2104 / RFC 8018) so the client does not
// depend on a system crypto library.
#include "vxaos_crypto.h"

#include <stdint.h>
#include <stdlib.h>
#include <string.h>

static const uint32_t K256[64] = {
  0x428a2f98u, 0x71374491u, 0xb5c0fbcfu, 0xe9b5dba5u,
  0x3956c25bu, 0x59f111f1u, 0x923f82a4u, 0xab1c5ed5u,
  0xd807aa98u, 0x12835b01u, 0x243185beu, 0x550c7dc3u,
  0x72be5d74u, 0x80deb1feu, 0x9bdc06a7u, 0xc19bf174u,
  0xe49b69c1u, 0xefbe4786u, 0x0fc19dc6u, 0x240ca1ccu,
  0x2de92c6fu, 0x4a7484aau, 0x5cb0a9dcu, 0x76f988dau,
  0x983e5152u, 0xa831c66du, 0xb00327c8u, 0xbf597fc7u,
  0xc6e00bf3u, 0xd5a79147u, 0x06ca6351u, 0x14292967u,
  0x27b70a85u, 0x2e1b2138u, 0x4d2c6dfcu, 0x53380d13u,
  0x650a7354u, 0x766a0abbu, 0x81c2c92eu, 0x92722c85u,
  0xa2bfe8a1u, 0xa81a664bu, 0xc24b8b70u, 0xc76c51a3u,
  0xd192e819u, 0xd6990624u, 0xf40e3585u, 0x106aa070u,
  0x19a4c116u, 0x1e376c08u, 0x2748774cu, 0x34b0bcb5u,
  0x391c0cb3u, 0x4ed8aa4au, 0x5b9cca4fu, 0x682e6ff3u,
  0x748f82eeu, 0x78a5636fu, 0x84c87814u, 0x8cc70208u,
  0x90befffau, 0xa4506cebu, 0xbef9a3f7u, 0xc67178f2u
};

static uint32_t rotr32(uint32_t x, unsigned n) {
  return (x >> n) | (x << (32 - n));
}

typedef struct {
  uint32_t h[8];
  uint64_t bits;
  unsigned char buf[64];
  size_t used;
} Sha256;

static void sha256_init(Sha256 *c) {
  c->h[0] = 0x6a09e667u;
  c->h[1] = 0xbb67ae85u;
  c->h[2] = 0x3c6ef372u;
  c->h[3] = 0xa54ff53au;
  c->h[4] = 0x510e527fu;
  c->h[5] = 0x9b05688cu;
  c->h[6] = 0x1f83d9abu;
  c->h[7] = 0x5be0cd19u;
  c->bits = 0;
  c->used = 0;
}

static void sha256_block(Sha256 *c, const unsigned char *p) {
  uint32_t w[64];
  uint32_t a, b, d, e, f, g, hh, cc;
  uint32_t s0, s1, t1, t2, ch, maj;
  int i;

  for (i = 0; i < 16; i++) {
    w[i] = ((uint32_t)p[i * 4] << 24) |
           ((uint32_t)p[i * 4 + 1] << 16) |
           ((uint32_t)p[i * 4 + 2] << 8) |
           ((uint32_t)p[i * 4 + 3]);
  }
  for (i = 16; i < 64; i++) {
    s0 = rotr32(w[i - 15], 7) ^ rotr32(w[i - 15], 18) ^ (w[i - 15] >> 3);
    s1 = rotr32(w[i - 2], 17) ^ rotr32(w[i - 2], 19) ^ (w[i - 2] >> 10);
    w[i] = w[i - 16] + s0 + w[i - 7] + s1;
  }
  a = c->h[0]; b = c->h[1]; cc = c->h[2]; d = c->h[3];
  e = c->h[4]; f = c->h[5]; g = c->h[6]; hh = c->h[7];
  for (i = 0; i < 64; i++) {
    s1 = rotr32(e, 6) ^ rotr32(e, 11) ^ rotr32(e, 25);
    ch = (e & f) ^ ((~e) & g);
    t1 = hh + s1 + ch + K256[i] + w[i];
    s0 = rotr32(a, 2) ^ rotr32(a, 13) ^ rotr32(a, 22);
    maj = (a & b) ^ (a & cc) ^ (b & cc);
    t2 = s0 + maj;
    hh = g; g = f; f = e; e = d + t1;
    d = cc; cc = b; b = a; a = t1 + t2;
  }
  c->h[0] += a; c->h[1] += b; c->h[2] += cc; c->h[3] += d;
  c->h[4] += e; c->h[5] += f; c->h[6] += g; c->h[7] += hh;
}

static void sha256_update(Sha256 *c, const unsigned char *data, size_t len) {
  size_t n;
  if (!data || len == 0) return;
  c->bits += (uint64_t)len * 8u;
  while (len > 0) {
    n = 64 - c->used;
    if (n > len) n = len;
    memcpy(c->buf + c->used, data, n);
    c->used += n;
    data += n;
    len -= n;
    if (c->used == 64) {
      sha256_block(c, c->buf);
      c->used = 0;
    }
  }
}

static void sha256_final(Sha256 *c, unsigned char *out) {
  uint64_t bits = c->bits;
  unsigned char pad = 0x80;
  unsigned char zero = 0;
  unsigned char lenbuf[8];
  int i;

  sha256_update(c, &pad, 1);
  while (c->used != 56) {
    sha256_update(c, &zero, 1);
  }
  for (i = 0; i < 8; i++) {
    lenbuf[7 - i] = (unsigned char)(bits >> (i * 8));
  }
  sha256_update(c, lenbuf, 8);
  for (i = 0; i < 8; i++) {
    out[i * 4] = (unsigned char)(c->h[i] >> 24);
    out[i * 4 + 1] = (unsigned char)(c->h[i] >> 16);
    out[i * 4 + 2] = (unsigned char)(c->h[i] >> 8);
    out[i * 4 + 3] = (unsigned char)(c->h[i]);
  }
}

void Crypto__sha256(const unsigned char *data, int len, unsigned char *out) {
  Sha256 c;
  if (!out) return;
  sha256_init(&c);
  if (data && len > 0) sha256_update(&c, data, (size_t)len);
  sha256_final(&c, out);
}

void Crypto__hmac_sha256(const unsigned char *key, int key_len,
                         const unsigned char *data, int data_len,
                         unsigned char *out) {
  unsigned char k[64];
  unsigned char ipad[64];
  unsigned char opad[64];
  unsigned char inner[32];
  Sha256 c;
  int i;

  if (!out) return;
  memset(k, 0, sizeof(k));
  if (key_len > 64) {
    Crypto__sha256(key, key_len, k);
  } else if (key && key_len > 0) {
    memcpy(k, key, (size_t)key_len);
  }
  for (i = 0; i < 64; i++) {
    ipad[i] = (unsigned char)(k[i] ^ 0x36);
    opad[i] = (unsigned char)(k[i] ^ 0x5c);
  }
  sha256_init(&c);
  sha256_update(&c, ipad, 64);
  if (data && data_len > 0) sha256_update(&c, data, (size_t)data_len);
  sha256_final(&c, inner);

  sha256_init(&c);
  sha256_update(&c, opad, 64);
  sha256_update(&c, inner, 32);
  sha256_final(&c, out);
}

void Crypto__pbkdf2_sha256(const unsigned char *password, int pass_len,
                           const unsigned char *salt, int salt_len,
                           int iterations, unsigned char *out, int out_len) {
  unsigned char u[32];
  unsigned char t[32];
  unsigned char *salt_block;
  unsigned char block[4];
  uint32_t counter;
  int blocks, i, j, k, take;

  if (!out || out_len <= 0) return;
  if (iterations < 1) iterations = 1;
  if (salt_len < 0) salt_len = 0;
  if (pass_len < 0) pass_len = 0;

  blocks = (out_len + 31) / 32;
  salt_block = (unsigned char *)malloc((size_t)salt_len + 4);
  if (!salt_block) {
    memset(out, 0, (size_t)out_len);
    return;
  }
  if (salt && salt_len > 0) memcpy(salt_block, salt, (size_t)salt_len);

  for (i = 1; i <= blocks; i++) {
    counter = (uint32_t)i;
    block[0] = (unsigned char)(counter >> 24);
    block[1] = (unsigned char)(counter >> 16);
    block[2] = (unsigned char)(counter >> 8);
    block[3] = (unsigned char)(counter);
    memcpy(salt_block + salt_len, block, 4);

    Crypto__hmac_sha256(password, pass_len, salt_block, salt_len + 4, u);
    memcpy(t, u, 32);
    for (j = 1; j < iterations; j++) {
      Crypto__hmac_sha256(password, pass_len, u, 32, u);
      for (k = 0; k < 32; k++) t[k] ^= u[k];
    }
    take = out_len - (i - 1) * 32;
    if (take > 32) take = 32;
    memcpy(out + (i - 1) * 32, t, (size_t)take);
  }
  free(salt_block);
}
