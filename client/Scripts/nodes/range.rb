class SRange < Control
  def initialize
    fail_to_init
  end

  def min_value
    @source.min_value
  end

  def min_value=(v)
    @source.min_value = v
  end

  def max_value
    @source.max_value
  end

  def max_value=(v)
    @source.max_value = v
  end

  def step
    @source.step
  end

  def step=(v)
    @source.step
  end

  def page
    @source.page
  end

  def page=(v)
    @source.page = v
  end

  def value
    @source.value
  end

  def value=(v)
    @source.value = v
  end

  def ratio
    @source.ratio
  end

  def ratio=(v)
    @source.ratio = v
  end

  def exp_edit
    @source.exp_edit
  end

  def exp_edit=(v)
    @source.exp_edit
  end

  def rounded
    @source.rounded
  end

  def rounded=(v)
    @source.rounded = v
  end

  def allow_greater
    @source.allow_greater
  end

  def allow_greater=(v)
    @source.allow_greater = v
  end

  def allow_lesser
    @source.allow_lesser
  end

  def allow_lesser=(v)
    @source.allow_lesser = v
  end

  def share(node)
    @source.share node.source
  end

  def unshare
    @source.unshare
  end
end