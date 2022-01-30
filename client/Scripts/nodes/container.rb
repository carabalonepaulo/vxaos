class Container < Control
  def initialize(source)
    if source.class.to_s == 'Godot::Container'
      @source = source
    else
      @source = $system.create_godot_object 'Container'
    end
    init_inner_control_vars
  end

  def fit_child_in_rect(control, rect)
    @source.fit_child_in_rect control.source, rect.source
  end

  def queue_sort
    @source.queue_sort
  end
end