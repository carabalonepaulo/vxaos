class VideoPlayer < Control
  def initialize(source = nil)
    init_source source
    init_inner_control_vars
  end

  def current_godot_object
    "VideoPlayer"
  end

  def audio_track
    @source.audio_track
  end

  def audio_track=(v)
    @source.audio_track = v
  end

  def stream
    @source.stream
  end

  def stream=(v)
    @source.stream = v
  end

  def volume_db
    @source.volume_db
  end

  def volume_db=(v)
    @source.volume_db = v
  end

  def volume
    @source.volume
  end

  def volume=(v)
    @source.volume = v
  end

  def autoplay
    @source.autoplay
  end

  def autoplay=(v)
    @source.autoplay = v
  end

  def paused
    @source.paused
  end

  def paused=(v)
    @source.paused = v
  end

  def expand
    @source.expand
  end

  def expand=(v)
    @source.expand = v
  end

  def buffering_msec
    @source.buffering_msec
  end

  def buffering_msec=(v)
    @source.buffering_msec = v
  end

  def stream_position
    @source.stream_position
  end

  def stream_position=(v)
    @source.stream_position = v
  end

  def bus
    @source.bus
  end

  def bus=(v)
    @source.bus = v
  end

  def get_stream_name
    @source.get_stream_name
  end

  def get_video_texture
    @source.get_video_texture
  end

  def is_playing?
    @source.is_playing
  end

  def play
    @source.play
  end

  def stop
    @source.stop
  end
end
