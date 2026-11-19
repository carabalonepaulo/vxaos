#==============================================================================
# ** Handle_Data
#------------------------------------------------------------------------------
#  Mensagens server -> client no protocolo v2.
#==============================================================================

module Handle_Data

  def handle_message(op, payload)
    debug_log("recv op=#{op} bytes=#{payload.bytesize}")
    case op
    when Protocol::Server::PONG
      return
    when Protocol::Server::AUTH_CHALLENGE
      handle_auth_challenge(payload)
    when Protocol::Server::AUTH_OK
      handle_auth_ok(payload)
    when Protocol::Server::AUTH_ERROR
      handle_auth_error(payload)
    when Protocol::Server::ENTER_WORLD
      handle_enter_world(payload)
    when Protocol::Server::CHARACTER_CREATED
      handle_character_created(payload)
    when Protocol::Server::CHARACTER_ERROR
      handle_character_error(payload)
    when Protocol::Server::CHARACTER_DELETED
      handle_character_deleted(payload)
    when Protocol::Server::VITALS
      handle_player_vitals(payload)
    when Protocol::Server::ITEM
      handle_player_item(payload)
    when Protocol::Server::GOLD
      handle_player_gold(payload)
    when Protocol::Server::PARAM
      handle_player_param(payload)
    when Protocol::Server::EQUIP
      handle_player_equip(payload)
    when Protocol::Server::SKILL
      handle_player_skill(payload)
    when Protocol::Server::POINTS
      handle_player_points(payload)
    when Protocol::Server::HOTBAR
      handle_player_hotbar(payload)
    when Protocol::Server::ATTACK_PLAYER
      handle_attack_player(payload)
    when Protocol::Server::ATTACK_ENEMY
      handle_attack_enemy(payload)
    when Protocol::Server::ANIMATION
      handle_animation(payload)
    when Protocol::Server::EVENT_DATA
      handle_event_data(payload)
    when Protocol::Server::EVENT_MOVE
      handle_event_move(payload)
    when Protocol::Server::ENEMY_REVIVE
      handle_enemy_revive(payload)
    when Protocol::Server::BALLOON
      handle_balloon(payload)
    when Protocol::Server::DROP_ADD
      handle_add_drop(payload)
    when Protocol::Server::DROP_REMOVE
      handle_remove_drop(payload)
    when Protocol::Server::ADMIN_COMMAND
      handle_admin_command(payload)
    when Protocol::Server::PROJECTILE
      handle_add_projectile(payload)
    when Protocol::Server::EXP
      handle_player_exp(payload)
    when Protocol::Server::TRANSFER
      handle_transfer_player(payload)
    when Protocol::Server::EVENT_COMMAND
      handle_event_command(payload)
    when Protocol::Server::SWITCH
      handle_switch(payload)
    when Protocol::Server::VARIABLE
      handle_variable(payload)
    when Protocol::Server::SELF_SWITCH
      handle_self_switch(payload)
    when Protocol::Server::GLOBAL_SWITCHES
      handle_global_switches(payload)
    when Protocol::Server::STATE
      handle_player_state(payload)
    when Protocol::Server::BUFF
      handle_player_buff(payload)
    when Protocol::Server::GUILD_CREATE_OPEN
      handle_guild_create_open
    when Protocol::Server::GUILD_OPEN
      handle_guild_open(payload)
    when Protocol::Server::GUILD_NAME
      handle_guild_name(payload)
    when Protocol::Server::GUILD_LEADER
      handle_guild_leader(payload)
    when Protocol::Server::GUILD_NOTICE
      handle_guild_notice(payload)
    when Protocol::Server::GUILD_MEMBER_REMOVE
      handle_guild_member_remove(payload)
    when Protocol::Server::REQUEST
      handle_request(payload)
    when Protocol::Server::PARTY_JOIN
      handle_party_join(payload)
    when Protocol::Server::PARTY_LEAVE
      handle_party_leave(payload)
    when Protocol::Server::PARTY_DISSOLVE
      handle_party_dissolve
    when Protocol::Server::FRIENDS
      handle_friends(payload)
    when Protocol::Server::FRIEND_ADD
      handle_friend_add(payload)
    when Protocol::Server::FRIEND_REMOVE
      handle_friend_remove(payload)
    when Protocol::Server::REQUEST_ACCEPT
      handle_request_accept(payload)
    when Protocol::Server::WINDOW_CLOSE
      handle_window_close
    when Protocol::Server::TRADE_ITEM
      handle_trade_item(payload)
    when Protocol::Server::TRADE_GOLD
      handle_trade_gold(payload)
    when Protocol::Server::QUEST_ADD
      handle_quest_add(payload)
    when Protocol::Server::QUEST_FINISH
      handle_quest_finish(payload)
    when Protocol::Server::VIP_DAYS
      handle_vip_days(payload)
    when Protocol::Server::SEX
      handle_player_sex(payload)
    when Protocol::Server::GRAPHIC
      handle_player_graphic(payload)
    when Protocol::Server::TELEPORT_OPEN
      handle_open_teleport(payload)
    when Protocol::Server::SHOP_OPEN
      handle_open_shop(payload)
    when Protocol::Server::BANK_OPEN
      handle_open_bank(payload)
    when Protocol::Server::BANK_ITEM
      handle_bank_item(payload)
    when Protocol::Server::BANK_GOLD
      handle_bank_gold(payload)
    when Protocol::Server::PLAYER_DATA
      handle_player_data(payload)
    when Protocol::Server::PLAYER_LEFT
      handle_player_left(payload)
    when Protocol::Server::PLAYER_MOVE
      handle_player_move(payload)
    when Protocol::Server::MAP_CHAT
      handle_map_chat(payload)
    when Protocol::Server::CHAT
      handle_chat(payload)
    when Protocol::Server::LOGOUT
      handle_logout
    when Protocol::Server::ERROR
      handle_error(payload)
    when Protocol::Server::DISCONNECT
      handle_disconnect(payload)
    end
  end

  def handle_auth_challenge(payload)
    r = ProtoReader.new(payload)
    salt = r.blob
    kdf = r.u8
    iterations = r.u32
    nonce = r.blob
    server_time = r.i64
    debug_log("challenge kdf=#{kdf} iterations=#{iterations} salt=#{salt.bytesize} nonce=#{nonce.bytesize}")
    if kdf != 1 || @auth_user.nil? || @auth_pass.nil?
      disconnect
      show_alert(Vocab::ConnectionFailed)
      return
    end
    verifier = Crypto.pbkdf2_sha256(@auth_pass, salt, iterations, 32)
    @auth_verifier = verifier
    @auth_salt = salt
    @auth_iterations = iterations
    @auth_nonce = nonce
    client_nonce = Crypto.random_bytes(16)
    proof = Crypto.login_proof(verifier, @auth_user, nonce, client_nonce)
    w = ProtoWriter.new
    w.blob(client_nonce)
    w.blob(proof)
    debug_log("sending auth proof for #{@auth_user}")
    send_frame(Protocol::Client::AUTH_PROOF, w.to_s)
  end

  def read_actor_summary(r)
    actor = Actor.new
    actor.name = r.string
    actor.character_name = r.string
    actor.character_index = r.u8
    actor.face_name = r.string
    actor.face_index = r.u8
    actor.sex = r.u8
    actor.equips = [$data_weapons[r.u16]]
    (Configs::MAX_EQUIPS - 1).times { actor.equips << $data_armors[r.u16] }
    actor
  end

  def handle_auth_ok(payload)
    r = ProtoReader.new(payload)
    @group = [r.u8, Enums::Group::ADMIN].min
    @vip_time = Time.at(r.i64)
    count = r.count
    @actors = {}
    count.times do
      slot = r.u8
      @actors[slot] = read_actor_summary(r)
    end
    debug_log("auth ok group=#{@group} actors=#{count}")
    $windows[:login].save_user
    SceneManager.goto(Scene_Character)
  end

  def show_alert(message)
    if $windows && $windows[:alert]
      $windows[:alert].show(message)
    else
      $alert_msg = message
    end
  end

  def report_error(message)
    if $windows && $windows[:chat]
      $windows[:chat].write_message(message, Configs::ERROR_COLOR)
    else
      show_alert(message)
    end
  end

  def handle_auth_error(payload)
    r = ProtoReader.new(payload)
    r.u16
    message = r.string
    message = Vocab::InvalidPass if message.empty?
    disconnect
    if SceneManager.scene_is?(Scene_Login)
      show_alert(message)
    else
      $alert_msg = message
      DataManager.back_login
    end
  end

  def handle_character_created(payload)
    r = ProtoReader.new(payload)
    slot = r.u8
    @actors[slot] = read_actor_summary(r)
    $windows[:create_char].hide if $windows[:create_char]
    $windows[:use_char].refresh if $windows[:use_char]
  end

  def handle_character_error(payload)
    r = ProtoReader.new(payload)
    r.u16
    message = r.string
    if $windows[:alert]
      $windows[:alert].show(message)
    else
      $alert_msg = message
    end
  end

  def handle_character_deleted(payload)
    r = ProtoReader.new(payload)
    slot = r.u8
    @actors.delete(slot)
    $windows[:use_char].refresh if $windows[:use_char]
  end

  def read_snapshot(r)
    s = {}
    s[:id] = r.u16
    s[:group] = r.u8
    s[:name] = r.string
    s[:character_name] = r.string
    s[:character_index] = r.u8
    s[:face_name] = r.string
    s[:face_index] = r.u8
    s[:sex] = r.u8
    s[:class_id] = r.u16
    s[:level] = r.i32
    s[:hp] = r.i32
    s[:mp] = r.i32
    s[:mhp] = r.i32
    s[:mmp] = r.i32
    s[:exp] = r.i32
    s[:points] = r.i32
    s[:gold] = r.i64
    s[:guild] = r.string
    s[:map_id] = r.u16
    s[:x] = r.u16
    s[:y] = r.u16
    s[:direction] = r.u8
    s[:equips] = []
    Configs::MAX_EQUIPS.times { s[:equips] << r.u16 }
    s
  end

  def read_items(r)
    count = r.count
    items = []
    count.times do
      items << [r.u8, r.u16, r.u16]
    end
    items
  end

  def handle_enter_world(payload)
    r = ProtoReader.new(payload)
    s = read_snapshot(r)
    params = []
    8.times { params << r.i32 }
    items = read_items(r)
    weapons = read_items(r)
    armors = read_items(r)
    skills = []
    r.count.times { skills << r.u16 }
    friends = []
    r.count.times { friends << r.string }
    quests = []
    r.count.times { quests << [r.u8, r.u8] }
    hotbar = []
    r.count.times { hotbar << [r.u8, r.u16] }
    switches = []
    r.count.times { switches << r.bool }
    variables = []
    r.count.times { variables << r.i32 }
    self_switches = []
    r.count.times { self_switches << [r.u16, r.u16, r.string, r.bool] }

    DataManager.create_game_objects
    $game_party.setup_starting_members
    @player_id = s[:id]
    $game_player.group = s[:group]
    $game_actors[1].name = s[:name]
    $game_actors[1].set_graphic(s[:character_name], s[:character_index], s[:face_name], s[:face_index])
    $game_actors[1].change_class(s[:class_id])
    $game_actors[1].sex = s[:sex]
    $game_actors[1].change_exp(s[:exp], false)
    $game_actors[1].clear_param_plus
    params.each_with_index do |value, param_id|
      $game_actors[1].add_param(param_id, value - $game_actors[1].param(param_id))
    end
    s[:equips].each_with_index { |item_id, slot_id| $game_actors[1].change_equip_by_id(slot_id, item_id) }
    $game_actors[1].hp = s[:hp]
    $game_actors[1].mp = s[:mp]
    $game_actors[1].points = s[:points]
    $game_actors[1].guild_name = s[:guild]
    debug_log("enter world points=#{$game_actors[1].points} hp=#{$game_actors[1].hp}/#{$game_actors[1].mhp}")
    $game_party.gain_gold(s[:gold])
    items.each { |entry| $game_party.gain_item($data_items[entry[1]], entry[2]) }
    weapons.each { |entry| $game_party.gain_item($data_weapons[entry[1]], entry[2]) }
    armors.each { |entry| $game_party.gain_item($data_armors[entry[1]], entry[2]) }
    skills.each { |skill_id| $game_actors[1].learn_skill(skill_id) }
    $game_actors[1].friends = friends
    quests.each { |entry| $game_actors[1].quests[entry[0]] = Game_Quest.new(entry[0], entry[1]) }
    hotbar.each_with_index { |entry, id| $game_actors[1].change_hotbar(id, entry[0], entry[1]) }
    switches.each_with_index { |value, switch_id| $game_switches[switch_id + 1] = value }
    variables.each_with_index { |value, variable_id| $game_variables[variable_id + 1] = value }
    self_switches.each { |entry| $game_self_switches[[entry[0], entry[1], entry[2]]] = entry[3] }

    $game_map.setup(s[:map_id])
    $game_player.moveto(s[:x], s[:y])
    $game_player.set_direction(s[:direction])
    $game_player.refresh
    Graphics.frame_count = 0
    RPG::BGM.stop
    RPG::BGS.stop
    RPG::ME.stop
    $game_map.autoplay
    @in_game = true
    $typing = nil
    $wait_player_move = false
    SceneManager.goto(Scene_Map)
  end

  def handle_player_vitals(payload)
    r = ProtoReader.new(payload)
    player_id = r.u16
    hp = r.i32
    mp = r.i32
    if @player_id == player_id
      dif_hp = hp - $game_actors[1].hp
      dif_mp = mp - $game_actors[1].mp
      $game_actors[1].hp = hp
      $game_actors[1].mp = mp
      $game_player.change_damage(dif_hp, dif_mp) unless dif_hp == 0 && dif_mp == 0
      $windows[:hud].refresh if $windows[:hud]
      $windows[:status].refresh if $windows[:status] && $windows[:status].visible
    elsif (player = $game_map.players[player_id])
      player.actor.hp = hp
      player.actor.mp = mp
      $windows[:target_hud].refresh if $windows[:target_hud] && $game_player.target == player
    end
  end

  def handle_player_item(payload)
    r = ProtoReader.new(payload)
    item_id = r.u16
    kind = r.u8
    amount = r.i32
    drop_sound = r.bool
    popup = r.bool
    item = $game_party.item_object(kind, item_id)
    return unless item
    $game_party.gain_item(item, amount)
    $windows[:item].refresh if $windows[:item] && $windows[:item].visible
    $windows[:hotbar].refresh if $windows[:hotbar] && $game_actors[1].hotbar.include?(item)
    $windows[:popup].show(item.name, item.icon_index, amount, item.is_a?(RPG::Item) && item.key_item?) if popup && $windows[:popup]
    Sound.play_ok if drop_sound
  end

  def handle_player_gold(payload)
    r = ProtoReader.new(payload)
    amount = r.i64
    shop_sound = r.bool
    popup = r.bool
    $game_party.gain_gold(amount)
    $windows[:item].refresh_gold_bar if $windows[:item] && $windows[:item].visible
    $windows[:shop].refresh if $windows[:shop] && $windows[:shop].visible
    $windows[:popup].show(Vocab.currency_unit, Configs::GOLD_ICON, amount) if popup && $windows[:popup]
    Sound.play_shop if shop_sound
  end

  def handle_player_param(payload)
    r = ProtoReader.new(payload)
    player_id = r.u16
    param_id = r.u8
    value = r.i32
    if @player_id == player_id
      $game_actors[1].add_param(param_id, value)
      $windows[:status].refresh if $windows[:status] && $windows[:status].visible
      $windows[:hud].refresh if param_id < 2 && $windows[:hud]
    elsif (player = $game_map.players[player_id])
      player.actor.add_param(param_id, value)
      $windows[:target_hud].refresh if param_id < 2 && $windows[:target_hud] && $game_player.target == player
    end
  end

  def handle_player_equip(payload)
    r = ProtoReader.new(payload)
    player_id = r.u16
    slot_id = r.u8
    item_id = r.u16
    if @player_id == player_id
      $game_actors[1].change_equip_by_id(slot_id, item_id)
      $windows[:equip].refresh if $windows[:equip]
      $windows[:hud].refresh if $windows[:hud]
      $windows[:status].refresh if $windows[:status] && $windows[:status].visible
      $windows[:skill].refresh if $windows[:skill] && $windows[:skill].visible
      $windows[:hotbar].refresh if $windows[:hotbar]
      Sound.play_equip
    elsif (player = $game_map.players[player_id])
      player.actor.change_equip_by_id(slot_id, item_id)
      $windows[:target_hud].refresh if $windows[:target_hud] && $game_player.target == player
    end
  end

  def handle_player_skill(payload)
    r = ProtoReader.new(payload)
    skill_id = r.u16
    learned = r.bool
    if learned
      $game_actors[1].learn_skill(skill_id)
    else
      $game_actors[1].forget_skill(skill_id)
    end
    $windows[:skill].refresh if $windows[:skill] && $windows[:skill].visible
    $windows[:hotbar].refresh if $windows[:hotbar] && $game_actors[1].hotbar.include?($data_skills[skill_id])
  end

  def handle_player_points(payload)
    r = ProtoReader.new(payload)
    $game_actors[1].points = r.i32
    $windows[:status].refresh if $windows[:status] && $windows[:status].visible
  end

  def handle_player_hotbar(payload)
    r = ProtoReader.new(payload)
    slot = r.u8
    type = r.u8
    item_id = r.u16
    $game_actors[1].change_hotbar(slot, type, item_id)
    $windows[:hotbar].refresh if $windows[:hotbar]
  end

  def handle_attack_player(payload)
    r = ProtoReader.new(payload)
    attacker_id = r.u16
    attacker_type = r.u8
    ani_index = r.u8
    player_id = r.u16
    hp_damage = r.i32
    mp_damage = r.i32
    critical = r.bool
    animation_id = r.u16
    not_show_missed = r.bool
    attacker = attacker_type == 1 ? $game_map.events[attacker_id] : (attacker_id == @player_id ? $game_player : $game_map.players[attacker_id])
    return unless attacker
    attacker.animate_attack(ani_index) if attacker.respond_to?(:animate_attack)
    if @player_id == player_id
      $game_actors[1].hp += hp_damage
      $game_actors[1].mp += mp_damage
      if $game_actors[1].dead?
        $game_player.die
      else
        $game_player.change_damage(hp_damage, mp_damage, critical, animation_id, not_show_missed)
        $game_map.screen.start_shake(4, 4, 20) if critical
      end
      $windows[:hud].refresh if $windows[:hud]
      $windows[:status].refresh if $windows[:status] && $windows[:status].visible
    elsif (player = $game_map.players[player_id])
      player.actor.hp += hp_damage
      player.actor.mp += mp_damage
      player.change_damage(hp_damage, mp_damage, critical, animation_id, not_show_missed) if $game_map.in_screen?(player) && !player.actor.dead?
      $windows[:target_hud].refresh if $windows[:target_hud] && $game_player.target == player && !player.actor.dead?
    end
  end

  def handle_attack_enemy(payload)
    r = ProtoReader.new(payload)
    attacker_id = r.u16
    attacker_type = r.u8
    ani_index = r.u8
    event_id = r.u16
    hp_damage = r.i32
    mp_damage = r.i32
    critical = r.bool
    animation_id = r.u16
    attacker = attacker_type == 1 ? $game_map.events[attacker_id] : (attacker_id == @player_id ? $game_player : $game_map.players[attacker_id])
    return unless attacker
    attacker.animate_attack(ani_index) if attacker.respond_to?(:animate_attack)
    event = $game_map.events[event_id]
    return unless event && event.actor
    event.actor.hp += hp_damage
    event.actor.mp += mp_damage
    event.change_damage(hp_damage, mp_damage, critical, animation_id) if $game_map.in_screen?(event)
    event.erase if event.actor.dead?
    $windows[:target_hud].refresh if $windows[:target_hud] && $game_player.target == event && event.actor? && !event.actor.dead?
  end

  def handle_animation(payload)
    r = ProtoReader.new(payload)
    attacker_id = r.u16
    attacker_type = r.u8
    ani_index = r.u8
    character_id = r.u16
    character_type = r.u8
    animation_id = r.u16
    attacker = attacker_type == 1 ? $game_map.events[attacker_id] : (attacker_id == @player_id ? $game_player : $game_map.players[attacker_id])
    attacker.animate_attack(ani_index) if attacker && attacker.respond_to?(:animate_attack)
    if character_type == 2
      event = $game_map.events[character_id]
      event.animation_id = animation_id if event && $game_map.in_screen?(event)
    elsif @player_id == character_id
      $game_player.animation_id = animation_id
    elsif (player = $game_map.players[character_id]) && $game_map.in_screen?(player)
      player.animation_id = animation_id
    end
  end

  def handle_event_data(payload)
    r = ProtoReader.new(payload)
    event_id = r.u16
    x = r.u16
    y = r.u16
    d = r.u8
    hp = r.i32
    event = $game_map.events[event_id]
    return unless event
    event.moveto(x, y) unless event.x == x && event.y == y
    event.set_direction(d) unless event.respond_to?(:fixed_movement?) && event.fixed_movement?
    if event.actor
      event.actor.hp = hp
      event.erase if event.actor.dead?
    end
  end

  def handle_event_move(payload)
    r = ProtoReader.new(payload)
    event_id = r.u16
    x = r.u16
    y = r.u16
    d = r.u8
    event = $game_map.events[event_id]
    event.move(x, y, d) if event
  end

  def handle_enemy_revive(payload)
    r = ProtoReader.new(payload)
    event_id = r.u16
    event = $game_map.events[event_id]
    return unless event
    event.erased = false
    event.refresh
    event.animation_id = 0
  end

  def handle_balloon(payload)
    r = ProtoReader.new(payload)
    character_id = r.u16
    character_type = r.u8
    balloon_id = r.u8
    if character_type == Enums::Target::ENEMY
      event = $game_map.events[character_id]
      event.balloon_id = balloon_id if event && $game_map.in_screen?(event)
    elsif @player_id == character_id
      $game_player.balloon_id = balloon_id
    elsif (player = $game_map.players[character_id]) && $game_map.in_screen?(player)
      player.balloon_id = balloon_id
    end
  end

  def handle_admin_command(payload)
    r = ProtoReader.new(payload)
    command = r.u8
    message = r.string
    case command
    when Enums::Command::BAN_IP
      $alert_msg = Vocab::IPBanned
    when Enums::Command::BAN_ACC
      $alert_msg = Vocab::AccBanned
    when Enums::Command::KICK
      $alert_msg = Vocab::Kicked
    when Enums::Command::MSG
      $admin_msg = message
    end
  end

  def handle_add_projectile(payload)
    r = ProtoReader.new(payload)
    start_x = r.u16
    start_y = r.u16
    finish_x = r.u16
    finish_y = r.u16
    target_x = r.u16
    target_y = r.u16
    projectile_type = r.u8
    projectile_id = r.u8
    return unless $game_map.in_screen?(start_x, start_y) || $game_map.in_screen?(finish_x, finish_y)
    config = projectile_type == Enums::Projectile::WEAPON ? Configs::RANGE_WEAPONS[projectile_id] : Configs::RANGE_SKILLS[projectile_id]
    return unless config
    projectile = Game_Projectile.new(start_x, start_y, finish_x, finish_y, target_x, target_y, config)
    $game_map.projectiles << projectile
    SceneManager.scene.add_projectile(projectile) if SceneManager.scene.respond_to?(:add_projectile)
  end

  def handle_add_drop(payload)
    r = ProtoReader.new(payload)
    item_id = r.u16
    kind = r.u8
    amount = r.u16
    x = r.u16
    y = r.u16
    item = $game_party.item_object(kind, item_id)
    return unless item
    drop = Game_Drop.new(item.name, item.icon_index, item.is_a?(RPG::Item) && item.key_item?, amount, x, y)
    $game_map.drops << drop
    SceneManager.scene.add_drop(drop) if SceneManager.scene.respond_to?(:add_drop)
  end

  def handle_remove_drop(payload)
    r = ProtoReader.new(payload)
    drop_id = r.u8
    return unless $game_map.drops[drop_id]
    $game_map.drops.delete_at(drop_id)
    SceneManager.scene.remove_drop(drop_id) if SceneManager.scene.respond_to?(:remove_drop)
  end

  def handle_player_exp(payload)
    r = ProtoReader.new(payload)
    player_id = r.u16
    amount = r.i32
    if @player_id == player_id
      $game_actors[1].result.exp = amount if amount > 0
      $game_actors[1].change_exp($game_actors[1].exp + amount, true)
      $windows[:hud].refresh if $windows[:hud]
    elsif (player = $game_map.players[player_id])
      player.actor.result.exp = amount if amount > 0 && $game_map.in_screen?(player)
      player.actor.change_exp(player.actor.exp + amount, false)
    end
  end

  def handle_transfer_player(payload)
    r = ProtoReader.new(payload)
    map_id = r.u16
    x = r.u16
    y = r.u16
    d = r.u8
    $game_player.reserve_transfer(map_id, x, y, d)
    $game_player.perform_transfer
  end

  def handle_event_command(payload)
    r = ProtoReader.new(payload)
    event_id = r.i32
    initial_index = r.u16
    final_index = r.u16
    if event_id > 0
      event = $game_map.events[event_id]
      return unless event
      list = event.list
      return if list.nil? || initial_index >= list.size
      final_index = list.size - 1 if final_index >= list.size
      commands = list[initial_index..final_index] + [list.last]
      if event.trigger == 4
        event.interpreter.setup(commands, event_id)
      else
        $game_map.interpreter.setup(commands, event_id)
      end
      debug_log("event cmd event=#{event_id} #{initial_index}..#{final_index} text=#{$game_message.has_text?} running=#{$game_map.interpreter.running?}")
    else
      common = $data_common_events[event_id.abs]
      return unless common
      list = common.list
      return if list.nil? || initial_index >= list.size
      final_index = list.size - 1 if final_index >= list.size
      $game_map.interpreter.setup(list[initial_index..final_index] + [list.last], event_id)
    end
  end

  def handle_switch(payload)
    r = ProtoReader.new(payload)
    $game_switches[r.u16] = r.bool
  end

  def handle_variable(payload)
    r = ProtoReader.new(payload)
    $game_variables[r.u16] = r.i32
  end

  def handle_self_switch(payload)
    r = ProtoReader.new(payload)
    key = [r.u16, r.u16, r.string]
    $game_self_switches[key] = r.bool
  end

  def handle_global_switches(payload)
    r = ProtoReader.new(payload)
    100.times do |i|
      $game_switches[Configs::MAX_PLAYER_SWITCHES + i + 1] = r.bool
    end
  end

  def handle_player_state(payload)
    r = ProtoReader.new(payload)
    state_id = r.u16
    if r.bool
      $game_actors[1].add_state(state_id)
    else
      $game_actors[1].remove_state(state_id)
    end
    refresh_states_window
  end

  def handle_player_buff(payload)
    r = ProtoReader.new(payload)
    param_id = r.u8
    level = r.i16
    if level == 1
      $game_actors[1].add_buff(param_id, 0)
    elsif level == -1
      $game_actors[1].add_debuff(param_id, 0)
    else
      $game_actors[1].remove_buff(param_id)
    end
    $windows[:hud].refresh if $windows[:hud] && param_id < 2
    $windows[:status].refresh if $windows[:status] && $windows[:status].visible
    refresh_states_window
  end

  def refresh_states_window
    return unless $windows[:states]
    actor = $game_actors[1]
    $windows[:states].visible = !actor.state_icons.empty? || !actor.buff_icons.empty?
    $windows[:states].refresh if $windows[:states].visible
  end

  def handle_guild_create_open
    $windows[:create_guild].show if $windows[:create_guild]
  end

  def handle_guild_open(payload)
    r = ProtoReader.new(payload)
    $game_guild.leader = r.string
    $game_guild.notice = r.string
    $game_guild.flag = []
    64.times { $game_guild.flag << r.u8 }
    size = r.u8
    $game_guild.online_size = r.u8
    $game_guild.members = []
    size.times { $game_guild.members << r.string }
    $windows[:guild].show if $windows[:guild]
  end

  def handle_guild_name(payload)
    r = ProtoReader.new(payload)
    name = r.string
    player_id = r.u16
    if @player_id == player_id
      $game_actors[1].guild_name = name
      if name.empty? && $windows[:guild]
        $windows[:guild].hide
        $windows[:guild].disable_buttons
      end
    elsif (player = $game_map.players[player_id])
      player.actor.guild_name = name
      player.refresh
    end
  end

  def handle_guild_leader(payload)
    r = ProtoReader.new(payload)
    $game_guild.leader = r.string
    $windows[:guild].refresh if $windows[:guild]
    $windows[:guild].disable_buttons if $windows[:guild]
  end

  def handle_guild_notice(payload)
    r = ProtoReader.new(payload)
    $game_guild.notice = r.string
    $windows[:guild].refresh if $windows[:guild]
  end

  def handle_guild_member_remove(payload)
    r = ProtoReader.new(payload)
    $game_guild.members.delete(r.string)
    $windows[:guild].refresh if $windows[:guild]
  end

  def handle_request(payload)
    r = ProtoReader.new(payload)
    _type = r.u8
    from_id = r.u16
    text = r.string
    $windows[:choice].show(text, Enums::Choice::REQUEST, from_id) if $windows[:choice]
  end

  def handle_party_join(payload)
    r = ProtoReader.new(payload)
    player_id = r.u16
    member = Game_NetPlayer.new(player_id)
    member.actor.name = r.string
    member.actor.set_graphic(r.string, r.u8, '', 0)
    member.actor.sex = r.u8
    Configs::MAX_EQUIPS.times { |slot_id| member.actor.change_equip_by_id(slot_id, r.u16) }
    member.actor.add_param(0, r.i32)
    member.actor.hp = r.i32
    member.actor.change_exp(r.i32, false)
    member.refresh
    $game_actors[1].party_members[player_id] = member
    if $windows[:chat]
      $windows[:chat].write_message("#{member.actor.name} #{Vocab::PartyMemberJoined}", Configs::SUCCESS_COLOR)
    end
    $windows[:party].show if $windows[:party]
  end

  def handle_party_leave(payload)
    r = ProtoReader.new(payload)
    player_id = r.u16
    member = $game_actors[1].party_members.delete(player_id)
    if member && $windows[:chat]
      $windows[:chat].write_message("#{member.actor.name} #{Vocab::PartyMemberLeave}", Configs::ERROR_COLOR)
    end
    $windows[:party].refresh if $windows[:party]
  end

  def handle_party_dissolve
    if $windows[:chat]
      $windows[:chat].write_message(Vocab::DissolvedParty, Configs::ERROR_COLOR)
    end
    $game_actors[1].party_members.clear
    $windows[:party].hide if $windows[:party]
  end

  def handle_friends(payload)
    r = ProtoReader.new(payload)
    size = r.u8
    online = []
    size.times { online << r.string }
    offline = $game_actors[1].friends - online
    $game_actors[1].friends = online + offline
    $game_actors[1].online_friends_size = size
    $windows[:friend].show if $windows[:friend]
  end

  def handle_friend_add(payload)
    r = ProtoReader.new(payload)
    name = r.string
    index = $game_actors[1].online_friends_size
    $game_actors[1].friends.insert(index, name)
    $game_actors[1].online_friends_size += 1
    $windows[:friend].refresh if $windows[:friend] && $windows[:friend].visible
    if $windows[:chat]
      $windows[:chat].write_message("#{name} #{Vocab::FriendAdded}", Configs::SUCCESS_COLOR)
    end
  end

  def handle_friend_remove(payload)
    r = ProtoReader.new(payload)
    index = r.u8
    $game_actors[1].friends.delete_at(index) if $game_actors[1].friends[index]
    $game_actors[1].online_friends_size -= 1 if index <= $game_actors[1].online_friends_size - 1
    $windows[:friend].refresh if $windows[:friend]
  end

  def handle_request_accept(payload)
    r = ProtoReader.new(payload)
    type = r.u8
    $game_trade.open if type == Enums::Request::TRADE && $game_trade
  end

  def handle_window_close
    $game_trade.close if $game_trade
    $game_bank.close if $game_bank
    if $windows[:shop] && $windows[:shop].visible
      $windows[:shop].hide
      $windows[:equip].hide if $windows[:equip]
      $windows[:amount].hide if $windows[:amount]
    end
    $windows[:teleport].hide if $windows[:teleport]
    $windows[:teleportinfo].hide if $windows[:teleportinfo]
    $windows[:create_guild].hide if $windows[:create_guild]
  end

  def handle_trade_item(payload)
    r = ProtoReader.new(payload)
    player_id = r.u16
    item_id = r.u16
    kind = r.u8
    amount = r.i16
    item = [nil, $data_items[item_id], $data_weapons[item_id], $data_armors[item_id]][kind]
    return unless item && $game_trade
    if @player_id == player_id
      $game_trade.gain_my_item(item, amount)
      $windows[:item].refresh if $windows[:item]
      $windows[:my_trade].refresh if $windows[:my_trade]
    else
      $game_trade.gain_his_item(item, amount)
      $windows[:his_trade].refresh if $windows[:his_trade]
    end
    $windows[:choice].hide if $windows[:choice] && $windows[:choice].finish_trade?
  end

  def handle_trade_gold(payload)
    r = ProtoReader.new(payload)
    player_id = r.u16
    amount = r.i32
    return unless $game_trade
    if @player_id == player_id
      $game_trade.gain_my_gold(amount)
      $windows[:item].refresh_gold_bar if $windows[:item]
      $windows[:my_trade].refresh_gold_bar if $windows[:my_trade]
    else
      $game_trade.gain_his_gold(amount)
      $windows[:his_trade].refresh_gold_bar if $windows[:his_trade]
    end
    $windows[:choice].hide if $windows[:choice] && $windows[:choice].finish_trade?
  end

  def handle_quest_add(payload)
    r = ProtoReader.new(payload)
    quest_id = r.u8
    return if quest_id >= Quests::DATA.size
    $game_actors[1].quests[quest_id] = Game_Quest.new(quest_id)
    $windows[:chat].write_message("#{Vocab::StartQuest} #{$game_actors[1].quests[quest_id].name}", Configs::SUCCESS_COLOR) if $windows[:chat]
    $windows[:quest].refresh if $windows[:quest] && $windows[:quest].visible
    $game_map.need_refresh = true
  end

  def handle_quest_finish(payload)
    r = ProtoReader.new(payload)
    quest_id = r.u8
    quest = $game_actors[1].quests[quest_id]
    return unless quest
    quest.state = Enums::Quest::FINISHED
    $windows[:chat].write_message("#{Vocab::FinishQuest} #{quest.name}", Configs::SUCCESS_COLOR) if $windows[:chat]
    if quest.repeat?
      $windows[:quest_info].hide if $windows[:quest_info] && $windows[:quest_info].quest == quest
      $game_actors[1].quests.delete(quest_id)
    end
    $windows[:quest].refresh if $windows[:quest] && $windows[:quest].visible
  end

  def handle_vip_days(payload)
    r = ProtoReader.new(payload)
    @vip_time = Time.at(r.i64)
  end

  def handle_player_sex(payload)
    r = ProtoReader.new(payload)
    player_id = r.u16
    sex = r.u8
    if @player_id == player_id
      $game_actors[1].sex = sex
    elsif (player = $game_map.players[player_id])
      player.actor.sex = sex
    end
  end

  def handle_player_graphic(payload)
    r = ProtoReader.new(payload)
    player_id = r.u16
    character_name = r.string
    character_index = r.u8
    face_name = r.string
    face_index = r.u8
    if @player_id == player_id
      $game_actors[1].set_graphic(character_name, character_index, face_name, face_index)
      $game_player.refresh
      $windows[:hud].refresh if $windows[:hud]
    elsif (player = $game_map.players[player_id])
      player.actor.set_graphic(character_name, character_index, face_name, face_index)
      player.refresh
    end
  end

  def handle_open_shop(payload)
    r = ProtoReader.new(payload)
    event_id = r.u16
    index = r.u16
    event = $game_map.events[event_id]
    return unless event
    list = event.list
    return if list.nil? || index >= list.size
    goods = [list[index].parameters]
    while index + 1 < list.size && list[index + 1].code == 605
      index += 1
      goods << list[index].parameters
    end
    $windows[:shop].show(goods, goods[0][4]) if $windows[:shop]
    $windows[:equip].show if $windows[:equip]
  end

  def handle_open_teleport(payload)
    r = ProtoReader.new(payload)
    id = r.u8
    $windows[:teleport].show(id) if $windows[:teleport]
  end

  def handle_open_bank(payload)
    r = ProtoReader.new(payload)
    $game_bank.gain_gold(r.i32)
    3.times do |kind|
      count = r.u8
      count.times do
        item_id = r.u16
        amount = r.u16
        item = [nil, $data_items[item_id], $data_weapons[item_id], $data_armors[item_id]][kind + 1]
        $game_bank.gain_item(item, amount) if item
      end
    end
    $game_bank.open
  end

  def handle_bank_item(payload)
    r = ProtoReader.new(payload)
    item_id = r.u16
    kind = r.u8
    amount = r.i16
    item = [nil, $data_items[item_id], $data_weapons[item_id], $data_armors[item_id]][kind]
    return unless item
    $game_bank.gain_item(item, amount)
    $windows[:bank].tab_page.index = kind - 1 if $windows[:bank]
    $windows[:bank].refresh if $windows[:bank]
  end

  def handle_bank_gold(payload)
    r = ProtoReader.new(payload)
    $game_bank.gain_gold(r.i32)
    $windows[:bank].refresh_gold_bar if $windows[:bank]
  end

  def handle_player_data(payload)
    r = ProtoReader.new(payload)
    s = read_snapshot(r)
    player = Game_NetPlayer.new(s[:id])
    player.group = s[:group]
    player.actor.name = s[:name]
    player.actor.set_graphic(s[:character_name], s[:character_index], '', 0)
    player.actor.sex = s[:sex]
    player.actor.change_class(s[:class_id])
    player.actor.change_exp(s[:exp], false)
    player.actor.clear_param_plus
    player.actor.add_param(0, s[:mhp] - player.actor.param_base(0))
    player.actor.add_param(1, s[:mmp] - player.actor.param_base(1))
    s[:equips].each_with_index { |item_id, slot_id| player.actor.change_equip_by_id(slot_id, item_id) }
    player.actor.hp = s[:hp]
    player.actor.mp = s[:mp]
    player.actor.guild_name = s[:guild]
    player.moveto(s[:x], s[:y])
    player.set_direction(s[:direction])
    player.refresh
    $game_map.players[s[:id]] = player
    SceneManager.scene.add_player(s[:id], player) if SceneManager.scene.respond_to?(:add_player)
    if $game_actors[1].party_members.has_key?(s[:id])
      $game_actors[1].party_members[s[:id]] = player
      $windows[:party].refresh
    end
  end

  def handle_player_left(payload)
    r = ProtoReader.new(payload)
    id = r.u16
    $game_map.players.delete(id)
    SceneManager.scene.remove_player(id) if SceneManager.scene.respond_to?(:remove_player)
    $windows[:party].refresh if $windows[:party] && $game_actors[1].party_members.has_key?(id)
  end

  def handle_player_move(payload)
    r = ProtoReader.new(payload)
    id = r.u16
    x = r.u16
    y = r.u16
    d = r.u8
    if @player_id == id
      $game_player.move(x, y, d)
      $wait_player_move = false
      $windows[:hud].change_opacity if $windows[:hud]
    elsif $game_map.players[id]
      $game_map.players[id].move(x, y, d)
    end
  end

  def handle_map_chat(payload)
    r = ProtoReader.new(payload)
    player_id = r.u16
    color_id = r.u8
    message = r.string
    return unless $windows[:chat]
    index = message.index(': ')
    return unless index
    name = message[0, index]
    msg = message[index + 2, message.size]
    return if $game_player.blocked.include?(name)
    $windows[:chat].write_message(message, color_id)
    player = @player_id == player_id ? $game_player : $game_map.players[player_id]
    if player && $game_map.in_screen?(player)
      player.message = $windows[:chat].word_wrap(msg, 217)
    end
  end

  def handle_chat(payload)
    r = ProtoReader.new(payload)
    color_id = r.u8
    message = r.string
    return unless $windows[:chat]
    index = message.index(': ')
    name = index ? message[0, index] : ''
    $windows[:chat].write_message(message, color_id) unless $game_player.blocked.include?(name)
  end

  def handle_logout
    @in_game = false
    RPG::BGM.stop
    RPG::BGS.stop
    RPG::ME.stop
    $data_system.title_bgm.play
    SceneManager.goto(Scene_Character)
  end

  def handle_error(payload)
    r = ProtoReader.new(payload)
    code = r.u16
    message = r.string
    debug_log("server error code=#{code} message=#{message}")
    report_error(message)
  end

  def handle_disconnect(payload)
    r = ProtoReader.new(payload)
    code = r.u16
    message = r.string
    debug_log("server disconnect code=#{code} message=#{message}")
    @in_game = false
    report_error(message)
    DataManager.back_login
  end

end
