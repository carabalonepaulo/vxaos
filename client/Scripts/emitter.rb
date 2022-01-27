class Emitter
  def initialize
    @callbacks = {}
  end
  
  def bind(signal, &callback)
    @callbacks[signal] ||= []
    @callbacks[signal].push callback
  end
  
  def emit(signal, *args)
    return if @callbacks[signal].nil?
    @callbacks[signal].each { |callback| callback.call(*args) }
  end
end