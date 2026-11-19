#==============================================================================
# ** Network
#------------------------------------------------------------------------------
#  Rede do cliente no protocolo v2: handshake, framing, autenticação e
#  despacho das mensagens.
#==============================================================================

class Network

  include Send_Data, Handle_Data

  attr_reader   :motd, :actors, :vip_time, :group, :player_id

  def initialize
    @group = Enums::Group::STANDARD
    @in_game = false
    @socket = nil
    @rx = ''.force_encoding('ASCII-8BIT')
    @handshake = false
    @pending_login = nil
    @auth_user = nil
    @auth_pass = nil
    @vip_time = Time.at(0)
    @motd = ''
    @actors = {}
    @player_id = -1
    @selected_actor_id = nil
  end

  def connect
    @socket = SocketLib.new(Configs::HOST, Configs::PORT)
    @rx = ''.force_encoding('ASCII-8BIT')
    @handshake = false
    @pending_login = nil
    debug_log("connected to #{Configs::HOST}:#{Configs::PORT}")
    send_hello
    true
  end

  def disconnect
    return unless connected?
    begin
      @socket.close
    rescue
    end
    @socket = nil
    @in_game = false
    @handshake = false
    @pending_login = nil
  end

  def connected?
    @socket
  end

  def standard_group?
    @group == Enums::Group::STANDARD
  end

  def vip?
    @vip_time && @vip_time > Time.now
  end

  def server_online?
    retry_count = 0
    successful = false
    while !successful && retry_count < 2
      begin
        connect
        successful = true
      rescue
        retry_count += 1
      end
    end
    successful
  end

  def update
    return unless @socket
    begin
      # eof? is true only when select() says the socket has no readable data;
      # loop while it is readable and drain the socket before parsing.
      while @socket && !@socket.eof?
        data = @socket.recv_non_block(8192)
        if data == -1
          disconnect
          $alert_msg ||= Vocab::ConnectionFailed
          DataManager.back_login unless SceneManager.scene_is?(Scene_Login)
          return
        end
        @rx << data
      end
      parse_stream
    rescue => e
      log_network_error(e)
      disconnect
      message = "Erro de rede: #{e.message}"
      if SceneManager.scene_is?(Scene_Login) && $windows && $windows[:alert]
        $windows[:alert].show(message)
      else
        $alert_msg = message
        DataManager.back_login unless SceneManager.scene_is?(Scene_Login)
      end
    end
  end

  # Writes network failures to System/network.log so a silent Script Error or
  # protocol mismatch can be diagnosed without the console.
  def log_network_error(error)
    File.open('System/network.log', 'a') do |file|
      file.puts("#{Time.now}: #{error.class}: #{error.message}")
      Array(error.backtrace).each { |line| file.puts("  #{line}") }
    end
  rescue
  end

  # Opt-in protocol trace for development (set VXAOS_DEBUG_NET=1).
  def debug_log(message)
    return unless ENV['VXAOS_DEBUG_NET']
    File.open('System/network.log', 'a') { |file| file.puts("#{Time.now}: #{message}") }
  rescue
  end

  def send_hello
    w = ProtoWriter.new
    w.u16(Protocol::VERSION)
    w.u16(Configs::GAME_VERSION)
    w.string("VXAOS #{Configs::GAME_VERSION}")
    @socket.send(Protocol::MAGIC + w.to_s)
  end

  def parse_stream
    unless @handshake
      parse_handshake
      return unless @handshake
    end
    loop do
      break if @rx.bytesize < 4
      size = @rx[0, 4].unpack('V')[0]
      if size < 2 || size > Protocol::MAX_FRAME
        disconnect
        $alert_msg = Vocab::ConnectionFailed
        DataManager.back_login unless SceneManager.scene_is?(Scene_Login)
        return
      end
      break if @rx.bytesize < 4 + size
      op = @rx[4, 2].unpack('v')[0]
      payload = @rx[6, size - 2]
      @rx = @rx[(4 + size)..-1] || ''.force_encoding('ASCII-8BIT')
      handle_message(op, payload)
      break unless @socket
    end
  end

  def parse_handshake
    debug_log("parse_handshake rx=#{@rx.bytesize}")
    return if @rx.bytesize < 27
    unless @rx[0, 8] == Protocol::MAGIC
      debug_log("bad handshake magic")
      disconnect
      $alert_msg = Vocab::ConnectionFailed
      DataManager.back_login unless SceneManager.scene_is?(Scene_Login)
      return
    end
    message_size = @rx[25, 2].unpack('v')[0]
    return if @rx.bytesize < 27 + message_size
    r = ProtoReader.new(@rx[8, 2 + 2 + 8 + 4 + 1 + 2 + message_size])
    proto = r.u16
    game = r.u16
    server_time = r.i64
    max_frame = r.u32
    status = r.u8
    message = r.string
    consumed = 8 + r.pos
    @rx = @rx[consumed..-1] || ''.force_encoding('ASCII-8BIT')
    if status != 0 || proto != Protocol::VERSION || game != Configs::GAME_VERSION
      debug_log("handshake rejected status=#{status} proto=#{proto} game=#{game}")
      disconnect
      $alert_msg = case status
      when 1 then Vocab::OldVersion
      when 2 then Vocab::ServerFull
      when 4 then Vocab::IPBanned
      else Vocab::ConnectionFailed
      end
      DataManager.back_login unless SceneManager.scene_is?(Scene_Login)
      return
    end
    @handshake = true
    debug_log("handshake ok")
    if @pending_login
      user, pass = @pending_login
      @pending_login = nil
      debug_log("sending auth for #{user}")
      send_auth_begin(user, pass)
    end
  end

  def send_login(user, pass)
    if @handshake
      send_auth_begin(user, pass)
    else
      @pending_login = [user, pass]
    end
  end

  def send_auth_begin(user, pass)
    @auth_user = user
    @auth_pass = pass
    debug_log("auth begin #{user} pass_len=#{pass.bytesize}")
    w = ProtoWriter.new
    w.string(user)
    send_frame(Protocol::Client::AUTH_BEGIN, w.to_s)
  end

  def send_frame(op, payload = ''.force_encoding('ASCII-8BIT'))
    return false unless @socket
    debug_log("send op=#{op} bytes=#{payload.bytesize}")
    @socket.send([payload.bytesize + 2].pack('V') + [op].pack('v') + payload)
    true
  end

end
