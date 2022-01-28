class Button < Control
  attr_accessor :source

  def initialize
    @source = $system.create_control 'Button'
    init_inner_control_vars
  end

  def text
    @source.text
  end

  def text=(v)
    @source.text = v.to_s
  end

  def disabled
  end

  def disabled=(v)
  end

  def toggle_mode
  end

  def toggle_mode=(v)
  end

  def shortcut_in_tooltip
  end

  def shortcut_in_tooltip=(v)
  end

  def pressed
  end

  def pressed=(v)
  end

  def action_mode
  end

  def action_mode=(v)
  end

  def button_mask
  end

  def button_mask=(v)
  end

  def enabled_focus_mode
  end

  def enabled_focus_mode=(v)
  end

  def keep_pressed_outside
  end

  def keep_pressed_outside=(v)
  end

  def focus_mode
  end

  def focus_mode=(v)
  end
end