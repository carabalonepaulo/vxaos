#==============================================================================
# ** Loading Screen
#------------------------------------------------------------------------------
#  Mostra a tela de carregamento durante a seleção de personagem sem atraso
#  artificial e sem barra de progresso. A seleção é enviada imediatamente ao
#  servidor; a tela permanece apenas enquanto o EnterWorld não chega.
#==============================================================================

class Scene_Character

  def loading?
    @loading == true
  end

  def begin_loading
    return if @loading
    @loading = true
    if Configs::LOADING_TITLES && !Configs::LOADING_TITLES.empty?
      change_background(Configs::LOADING_TITLES[rand(Configs::LOADING_TITLES.size)])
    end
    @config_icon.visible = false if @config_icon
    @loading_bar.visible = false if @loading_bar
    $windows[:vip_time].hide if $windows[:vip_time]
    $windows[:password].hide if $windows[:password]
    $windows[:config].hide if $windows[:config]
    $windows[:use_char].hide if $windows[:use_char]
  end

  def cancel_loading
    return unless @loading
    @loading = false
    refresh_sprite1
    @config_icon.visible = true if @config_icon
    $windows[:vip_time].show if $windows[:vip_time]
    $windows[:use_char].show if $windows[:use_char]
    $windows[:use_char].refresh if $windows[:use_char]
  end

  def update_all_windows
    super
    @config_icon.update if @config_icon
    # A barra existe apenas para compatibilidade; não há progresso real.
    @loading_bar.visible = false if @loading_bar
  end

end

class Window_UseChar

  def use_actor
    if SceneManager.scene.respond_to?(:loading?) && SceneManager.scene.loading?
      return
    end
    SceneManager.scene.begin_loading if SceneManager.scene.respond_to?(:begin_loading)
    $network.send_use_actor($actor_id)
  end

end
