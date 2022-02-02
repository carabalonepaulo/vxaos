class ScrollBar < Range
  def custom_step
    @source.custom_step
  end

  def custom_step=(v)
    @source.custom_step = v
  end
end