class ScrollBar < SRange
  def custom_step
    @source.custom_step
  end

  def current_godot_object
    "ScrollBar"
  end

  def custom_step=(v)
    @source.custom_step = v
  end
end