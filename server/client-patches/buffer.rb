#==============================================================================
# ** Buffer
#------------------------------------------------------------------------------
#  Codec binário do protocolo v2. Todos os inteiros são little-endian.
#==============================================================================

class ProtoWriter

  def initialize
    @buf = ''.force_encoding('ASCII-8BIT')
  end

  def u8(value)
    @buf << [value & 0xFF].pack('C')
    self
  end

  def bool(value)
    u8(value ? 1 : 0)
  end

  def u16(value)
    @buf << [value & 0xFFFF].pack('v')
    self
  end

  def i16(value)
    @buf << [value & 0xFFFF].pack('v')
    self
  end

  def u32(value)
    @buf << [value & 0xFFFFFFFF].pack('V')
    self
  end

  def i32(value)
    @buf << [value].pack('i')
    self
  end

  def i64(value)
    @buf << [value].pack('q')
    self
  end

  def string(value)
    b = value.to_s.force_encoding('ASCII-8BIT')
    raise 'Protocol string too long' if b.bytesize > 65535
    u16(b.bytesize)
    @buf << b
    self
  end

  def blob(value)
    b = value.to_s.force_encoding('ASCII-8BIT')
    raise 'Protocol blob too long' if b.bytesize > 65535
    u16(b.bytesize)
    @buf << b
    self
  end

  def count(value)
    u16(value)
  end

  def to_s
    @buf
  end

end

#==============================================================================
# ** ProtoReader
#==============================================================================

class ProtoReader

  attr_reader :pos

  def initialize(data)
    @data = data.dup.force_encoding('ASCII-8BIT')
    @pos = 0
  end

  def remaining
    @data.bytesize - @pos
  end

  def eof?
    @pos >= @data.bytesize
  end

  def u8
    raise 'Protocol short read' if @pos + 1 > @data.bytesize
    value = @data[@pos, 1].unpack('C')[0]
    @pos += 1
    value
  end

  def bool
    u8 == 1
  end

  def u16
    raise 'Protocol short read' if @pos + 2 > @data.bytesize
    value = @data[@pos, 2].unpack('v')[0]
    @pos += 2
    value
  end

  def i16
    value = u16
    value >= 0x8000 ? value - 0x10000 : value
  end

  def u32
    raise 'Protocol short read' if @pos + 4 > @data.bytesize
    value = @data[@pos, 4].unpack('V')[0]
    @pos += 4
    value
  end

  def i32
    raise 'Protocol short read' if @pos + 4 > @data.bytesize
    value = @data[@pos, 4].unpack('i')[0]
    @pos += 4
    value
  end

  def i64
    raise 'Protocol short read' if @pos + 8 > @data.bytesize
    value = @data[@pos, 8].unpack('q')[0]
    @pos += 8
    value
  end

  def string
    size = u16
    raise 'Protocol short string' if @pos + size > @data.bytesize
    value = @data[@pos, size]
    @pos += size
    value.force_encoding('UTF-8')
  end

  def blob
    size = u16
    raise 'Protocol short blob' if @pos + size > @data.bytesize
    value = @data[@pos, size]
    @pos += size
    value.force_encoding('ASCII-8BIT')
  end

  def count
    u16
  end

end
