#==============================================================================
# ** Send_Data
#------------------------------------------------------------------------------
#  Mensagens client -> server no protocolo v2. Somente o slice atual (login,
#  seleção de personagem, movimento, chat e logout) é enviado; as demais
#  funções são mantidas como no-op para a UI não enviar bytes do protocolo v1.
#==============================================================================

module Send_Data

  def send_create_account(user, pass, email)
    message = 'Registro pelo cliente desativado nesta versão.'
    $windows[:alert] ? $windows[:alert].show(message) : $alert_msg = message
  end

  # A janela de criação usa a ordem HP, MP, ATK, AGI, DEF, MAT, MDF, LUK.
  # O protocolo usa a ordem dos parâmetros do RPG Maker (HP, MP, ATK, DEF,
  # MAT, MDF, AGI, LUK).
  PARAM_TO_WIRE = [0, 1, 2, 4, 5, 6, 3, 7]

  def send_create_actor(actor_id, name, character_index, class_id, sex, params)
    return unless @socket
    w = ProtoWriter.new
    w.u8(actor_id)
    w.string(name)
    w.u8(character_index)
    w.u16(class_id)
    w.u8(sex)
    PARAM_TO_WIRE.each { |ui_index| w.u8(params[ui_index] || 0) }
    send_frame(Protocol::Client::CREATE_CHARACTER, w.to_s)
  end

  def send_remove_actor(actor_id, pass)
    return unless @socket
    actor = @actors[actor_id]
    return unless actor
    unless @auth_verifier && @auth_salt && @auth_nonce
      $windows[:alert].show(Vocab::InvalidPass) if $windows[:alert]
      debug_log("delete: missing auth material")
      return
    end
    entered = Crypto.pbkdf2_sha256(pass.to_s, @auth_salt, @auth_iterations, 32)
    unless entered == @auth_verifier
      $windows[:alert].show(Vocab::InvalidPass) if $windows[:alert]
      debug_log("delete: password mismatch len=#{pass.to_s.bytesize} actor=#{actor.name}")
      return
    end
    client_nonce = Crypto.random_bytes(16)
    proof = Crypto.delete_proof(@auth_verifier, @auth_user, @auth_nonce, client_nonce, actor.name)
    debug_log("delete: proof sent slot=#{actor_id} actor=#{actor.name}")
    w = ProtoWriter.new
    w.u8(actor_id)
    w.blob(client_nonce)
    w.blob(proof)
    send_frame(Protocol::Client::DELETE_CHARACTER, w.to_s)
  end

  def send_use_actor(actor_id)
    return unless @socket
    @selected_actor_id = actor_id
    w = ProtoWriter.new
    w.u8(actor_id)
    send_frame(Protocol::Client::SELECT_CHARACTER, w.to_s)
  end

  def send_player_movement(direction)
    return unless @socket
    w = ProtoWriter.new
    w.u8(direction)
    send_frame(Protocol::Client::MOVE, w.to_s)
  end

  def send_chat_message(message, talk_type, player_name = '')
    return unless @socket
    w = ProtoWriter.new
    w.u8(talk_type)
    w.string(message)
    w.string(player_name)
    send_frame(Protocol::Client::CHAT, w.to_s)
  end

  def send_logout
    return unless @socket
    send_frame(Protocol::Client::LOGOUT, ''.force_encoding('ASCII-8BIT'))
  end

  #--------------------------------------------------------------------------
  # Ações ainda não portadas para o slice atual.
  #--------------------------------------------------------------------------
  def send_player_attack
    return unless @socket
    send_frame(Protocol::Client::ATTACK, ''.force_encoding('ASCII-8BIT'))
  end

  def send_use_item(item_id)
    return unless @socket
    w = ProtoWriter.new
    w.u16(item_id)
    send_frame(Protocol::Client::USE_ITEM, w.to_s)
  end

  def send_use_skill(skill_id)
    return unless @socket
    w = ProtoWriter.new
    w.u16(skill_id)
    send_frame(Protocol::Client::USE_SKILL, w.to_s)
  end

  def send_balloon(balloon_id); end

  def send_use_hotbar(id)
    return unless @socket
    w = ProtoWriter.new
    w.u8(id)
    send_frame(Protocol::Client::USE_HOTBAR, w.to_s)
  end

  def send_add_drop(item_id, kind, amount)
    return unless @socket
    w = ProtoWriter.new
    w.u16(item_id)
    w.u8(kind)
    w.u16(amount)
    send_frame(Protocol::Client::DROP_ITEM, w.to_s)
  end

  def send_remove_drop(drop_id)
    return unless @socket
    server_id = @drop_ids ? (@drop_ids[drop_id] || drop_id) : drop_id
    w = ProtoWriter.new
    w.u32(server_id)
    send_frame(Protocol::Client::PICKUP_DROP, w.to_s)
  end

  def send_add_param(param_id)
    return unless @socket
    w = ProtoWriter.new
    w.u8(param_id)
    send_frame(Protocol::Client::SPEND_POINT, w.to_s)
  end

  def send_player_equip(item_id, slot_id)
    return unless @socket
    w = ProtoWriter.new
    w.u8(slot_id)
    w.u16(item_id)
    send_frame(Protocol::Client::EQUIP, w.to_s)
  end

  def send_player_hotbar(id, type, item_id)
    return unless @socket
    w = ProtoWriter.new
    w.u8(id)
    w.u8(type)
    w.u16(item_id)
    send_frame(Protocol::Client::SET_HOTBAR, w.to_s)
  end

  def send_target(target_id, type)
    return unless @socket
    w = ProtoWriter.new
    w.u8(type)
    w.u16(target_id)
    send_frame(Protocol::Client::SET_TARGET, w.to_s)
  end
  def send_open_friend_window
    return unless @socket
    send_frame(Protocol::Client::OPEN_FRIENDS, ''.force_encoding('ASCII-8BIT'))
  end

  def send_remove_friend(index)
    return unless @socket
    w = ProtoWriter.new
    w.u8(index)
    send_frame(Protocol::Client::REMOVE_FRIEND, w.to_s)
  end

  def send_create_guild(name, flag)
    return unless @socket
    w = ProtoWriter.new
    w.string(name)
    64.times { |i| w.u8(i < flag.size ? flag[i] : 255) }
    send_frame(Protocol::Client::GUILD_CREATE, w.to_s)
  end

  def send_open_guild_window
    return unless @socket
    send_frame(Protocol::Client::GUILD_OPEN, ''.force_encoding('ASCII-8BIT'))
  end

  def send_guild_leader(name)
    return unless @socket
    w = ProtoWriter.new
    w.string(name)
    send_frame(Protocol::Client::GUILD_SET_LEADER, w.to_s)
  end

  def send_change_guild_leader(name)
    send_guild_leader(name)
  end

  def send_guild_notice(notice)
    return unless @socket
    w = ProtoWriter.new
    w.string(notice)
    send_frame(Protocol::Client::GUILD_SET_NOTICE, w.to_s)
  end

  def send_remove_guild_member(name)
    return unless @socket
    w = ProtoWriter.new
    w.string(name)
    send_frame(Protocol::Client::GUILD_KICK, w.to_s)
  end

  def send_guild_request(name)
    return unless @socket
    w = ProtoWriter.new
    w.string(name)
    send_frame(Protocol::Client::GUILD_INVITE, w.to_s)
  end

  def send_leave_guild
    return unless @socket
    send_frame(Protocol::Client::GUILD_LEAVE, ''.force_encoding('ASCII-8BIT'))
  end

  def send_leave_party
    return unless @socket
    send_frame(Protocol::Client::PARTY_LEAVE, ''.force_encoding('ASCII-8BIT'))
  end

  def send_choice(index)
    return unless @socket
    w = ProtoWriter.new
    w.i32(index)
    send_frame(Protocol::Client::CHOICE, w.to_s)
  end

  def send_bank_item(item_id, kind, amount)
    return unless @socket
    w = ProtoWriter.new
    w.u16(item_id)
    w.u8(kind)
    w.i16(amount)
    send_frame(Protocol::Client::BANK_ITEM, w.to_s)
  end

  def send_bank_gold(amount)
    return unless @socket
    w = ProtoWriter.new
    w.i32(amount)
    send_frame(Protocol::Client::BANK_GOLD, w.to_s)
  end

  def send_close_window
    return unless @socket
    send_frame(Protocol::Client::CLOSE_WINDOW, ''.force_encoding('ASCII-8BIT'))
  end

  def send_buy_item(index, amount)
    return unless @socket
    w = ProtoWriter.new
    w.u8(index)
    w.u16(amount)
    send_frame(Protocol::Client::SHOP_BUY, w.to_s)
  end

  def send_sell_item(item_id, kind, amount)
    return unless @socket
    w = ProtoWriter.new
    w.u16(item_id)
    w.u8(kind)
    w.u16(amount)
    send_frame(Protocol::Client::SHOP_SELL, w.to_s)
  end
  def send_choice_telepot(index)
    return unless @socket
    w = ProtoWriter.new
    w.u8(index)
    send_frame(Protocol::Client::CHOICE_TELEPORT, w.to_s)
  end

  def send_next_event_command
    return unless @socket
    send_frame(Protocol::Client::EVENT_NEXT, ''.force_encoding('ASCII-8BIT'))
  end
  def send_request(type, player_id = -1)
    return unless @socket
    w = ProtoWriter.new
    w.u8(type)
    w.u16(player_id < 0 ? 65535 : player_id)
    send_frame(Protocol::Client::REQUEST, w.to_s)
  end

  def send_accept_request
    return unless @socket
    send_frame(Protocol::Client::REQUEST_ACCEPT, ''.force_encoding('ASCII-8BIT'))
  end

  def send_decline_request
    return unless @socket
    send_frame(Protocol::Client::REQUEST_DECLINE, ''.force_encoding('ASCII-8BIT'))
  end
  def send_trade_item(item_id, kind, amount)
    return unless @socket
    w = ProtoWriter.new
    w.u16(item_id)
    w.u8(kind)
    w.i16(amount)
    send_frame(Protocol::Client::TRADE_ITEM, w.to_s)
  end

  def send_trade_gold(amount)
    return unless @socket
    w = ProtoWriter.new
    w.i32(amount)
    send_frame(Protocol::Client::TRADE_GOLD, w.to_s)
  end
  def send_admin_command(command, str, int1 = 0, int2 = 0, int3 = 0)
    return unless @socket
    w = ProtoWriter.new
    w.u8(command)
    w.string(str)
    w.i32(int1)
    w.i32(int2)
    w.i16(int3)
    send_frame(Protocol::Client::ADMIN_COMMAND, w.to_s)
  end

end
