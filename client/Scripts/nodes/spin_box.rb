class SpinBox < SRange
  def initialize(source = nil)
    init_source source
    init_inner_control_vars
  end

  def align
    @source.align
  end

  def align=(v)
    @source.align = v
  end

  def editable
    @source.editable
  end

  def editable=(v)
    @source.editable = v
  end

  def prefix
    @source.prefix
  end

  def prefix=(v)
    @source.prefix = v
  end

  def suffix
    @source.suffix
  end

  def suffix=(v)
    @source.suffix = v
  end

  def apply
    @source.apply
  end

  def get_line_edit
    # TODO: LineEdit.new @source.get_line_edit
  end
end