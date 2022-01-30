class BaseButton < Control
  def initialize
    raise "Can't create instance of BaseButton."
  end

  def disabled
    @source.disabled
  end

  def disabled=(v)
    @source.disabled = v
  end

  def toggle_mode
    @source.toggle_mode
  end

  def toggle_mode=(v)
    @source.toggle_mode = v
  end

  def shortcut_in_tooltip
    @source.shortcut_in_tooltip
  end

  def shortcut_in_tooltip=(v)
    @source.shortcut_in_tooltip = v
  end

  def pressed
    @source.pressed
  end

  def pressed=(v)
    @source.pressed = v
  end

  def action_mode
    @source.action_mode
  end

  def action_mode=(v)
    @source.action_mode = v
  end

  def button_mask
    @source.button_mask
  end

  def button_mask=(v)
    @source.button_mask = v
  end

  def enabled_focus_mode
    @source.enabled_focus_mode
  end

  def enabled_focus_mode=(v)
    @source.enabled_focus_mode = v
  end

  def keep_pressed_outside
    @source.keep_pressed_outside
  end

  def keep_pressed_outside=(v)
    @source.keep_pressed_outside = v
  end

  def shortcut
    raise 'not implemented yet'
  end

  def shortcut=(v)
    raise 'not implemented yet'
  end

  def group
    raise 'not implemented yet'
  end

  def group=(v)
    raise 'not implemented yet'
  end

  def focus_mode
    @source.focus_mode
  end

  def focus_mode=(v)
    @source.focus_mode = v
  end

  def _pressed
  end

  def _toggled(button_pressed)
  end

  def get_draw_mode
    @source.get_draw_mode
  end

  def is_hovered?
    @source.is_hovered
  end

  def set_pressed_no_signal(pressed)
    @source.set_pressed_no_signal pressed
  end
end