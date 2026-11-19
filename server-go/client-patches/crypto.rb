#==============================================================================
# ** Crypto
#------------------------------------------------------------------------------
#  Wrappers para SHA-256/HMAC/PBKDF2 da VXAOS.dll. O servidor usa os mesmos
#  parâmetros no challenge-response.
#==============================================================================

module Crypto

  PBKDF2 = c_function('System/VXAOS.dll',
    'void Crypto__pbkdf2_sha256(const char *, int, const char *, int, int, char *, int)')
  HMAC = c_function('System/VXAOS.dll',
    'void Crypto__hmac_sha256(const char *, int, const char *, int, char *)')
  SHA256 = c_function('System/VXAOS.dll',
    'void Crypto__sha256(const char *, int, char *)')

  def self.pbkdf2_sha256(password, salt, iterations, key_len = 32)
    out = DL::CPtr.new(DL.malloc(key_len), key_len)
    PBKDF2.call(password, password.bytesize, salt, salt.bytesize, iterations, out, key_len)
    out.to_str
  end

  def self.hmac_sha256(key, data)
    out = DL::CPtr.new(DL.malloc(32), 32)
    HMAC.call(key, key.bytesize, data, data.bytesize, out)
    out.to_str
  end

  def self.sha256(data)
    out = DL::CPtr.new(DL.malloc(32), 32)
    SHA256.call(data, data.bytesize, out)
    out.to_str
  end

  def self.field(value)
    b = value.to_s.force_encoding('ASCII-8BIT')
    [b.bytesize].pack('V') + b
  end

  def self.login_proof(verifier, username, nonce, client_nonce)
    data = 'vxaos-login-v1'.force_encoding('ASCII-8BIT')
    data << field(username) << field(nonce) << field(client_nonce)
    hmac_sha256(verifier, data)
  end

  def self.delete_proof(verifier, username, login_nonce, client_nonce, character_name)
    data = 'vxaos-delete-v1'.force_encoding('ASCII-8BIT')
    data << field(username) << field(login_nonce) << field(client_nonce) << field(character_name)
    hmac_sha256(verifier, data)
  end

  def self.random_bytes(count)
    seed = Time.now.to_i ^ (object_id << 1)
    rng = Random.new(seed)
    Array.new(count) { rng.rand(256) }.pack('C*')
  end

end
