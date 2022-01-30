class Container < Control
  def initialize(source)
    raise "Can't create instance of Container."
  end

  def fit_child_in_rect(control, rect)
    @source.fit_child_in_rect control.source, rect.source
  end

  def queue_sort
    @source.queue_sort
  end
end