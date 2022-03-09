class BoxContainer < Container
  def initialize
    fail_to_init
  end

  def current_godot_object
    "BoxContainer"
  end

  def alignment
    @source.alignment
  end

  def alignment=(v)
    @source.alignment = v
  end

  def mouse_filter
    @source.mouse_filter
  end

  def mouse_filter=(v)
    @source.mouse_filter = v
  end

  def add_spacer(_begin)
    @source.add_spacer _begin
  end
end