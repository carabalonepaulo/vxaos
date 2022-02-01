class MarginContainer < Container
  def initialize(source = nil)
    init_source source
    init_inner_control_vars
  end

  def margin_right
    @source.margin_right
  end

  def margin_right=(v)
    @source.margin_right = v
  end

  def margin_top
    @source.margin_top
  end

  def margin_top=(v)
    @source.margin_top = v
  end

  def margin_left
    @source.margin_left
  end

  def margin_left=(v)
    @source.margin_left = v
  end

  def margin_bottom
    @source.margin_bottom
  end

  def margin_bottom=(v)
    @source.margin_bottom = v
  end
end