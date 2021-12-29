class Main
  def initialize(listener)
    setup listener
  end

  def setup(listener)
    @listener = listener
    @listener.on(:connected) { |cid| on_connected cid }
    @listener.on(:disconnected) { |cid| on_disconnected cid }
    @listener.on(:data) { |cid, buff| on_data cid, buff }
  end

  def on_connected(cid)
  end

  def on_disconnected(cid)
  end

  def on_data(cid, buff)
  end

  def update(dt)
  end
end