class SplitContainer < Container
  def initialize
    fail_to_init
  end

  def split_offset
    @source.split_offset
  end

  def split_offset=(v)
    @source.split_offset = v
  end

  def collapsed
    @source.collapsed
  end

  def collapsed=(v)
    @source.collapsed = v
  end

  def dragger_visibility
    @source.dragger_visibility
  end

  def dragger_visibility=(v)
    @source.dragger_visibility = v
  end

  def clamp_split_offset
    @source.clamp_split_offset
  end
end
