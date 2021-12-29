class Emitter
  def initialize
    @signals = {}
  end

  def emit(signal, *args)
    return unless @signals.has_key signal
    @signals[signal].each { |callback| callback.call *args }
  end

  def on(signal, &callback)
    @signals[signal] = [] unless @signals.has_key? signal
    @signals[signal] << callback
  end
end