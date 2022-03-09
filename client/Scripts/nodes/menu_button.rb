class MenuButton < Button
  def initialize(source = nil)
    init_source source
    init_inner_control_vars
  end

  def current_godot_object
    "MenuButton"
  end

  def switch_on_hover
    @source.switch_on_hover
  end

  def switch_on_hover=(v)
    @source.switch_on_hover = v
  end

  def flat
    @source.flat
  end

  def flat=(v)
    @source.flat = v
  end

  def toggle_mode
    @source.toggle_mode
  end

  def toggle_mode=(v)
    @source.toggle_mode = v
  end

  def action_mode
    @source.action_mode
  end

  def action_mode=(v)
    @source.action_mode = v
  end

  def focus_mode
    @source.focus_mode
  end

  def focus_mode=(v)
    @source.focus_mode = v
  end

  def get_popup
    # TODO: PopupMenu.new @source.get_popup
  end

  def set_disabled_shortcuts(disabled)
    @source.set_disabled_shortcuts disabled
  end
end