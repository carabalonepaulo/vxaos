class Button < BaseButton
  def initialize(source = nil)
    init_source source, 'Button'
    init_inner_control_vars
  end

  def init_inner_control_vars
    super
    @texture = nil
  end

  def text
    @source.text
  end

  def text=(v)
    @source.text = v.to_s
  end

  def icon
    @texture
  end

  def icon=(v)
    raise 'not implemented yet'
  end

  def flat
    @source.flat
  end

  def flat=(v)
    @source.flat = v
  end

  def clip_text
    @source.clip_text
  end

  def clip_text=(v)
    @source.clip_text = v
  end

  def align
    @source.align
  end

  def align=(v)
    @source.align = v
  end

  def expand_icon
    @source.expand_icon
  end

  def expand_icon=(v)
    @source.expand_icon = v
  end
end