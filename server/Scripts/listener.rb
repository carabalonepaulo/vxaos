class Listener < Emitter
  def initialize(send)
    @send = send
  end

  def send(cid, buff)
  end
end