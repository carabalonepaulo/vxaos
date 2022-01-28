module FileSystem
  def self.init(api)
    @@api = api
  end

  def self.exists(path)
    @@api.exists path
  end

  def self.read_all_text(path)
    @@api.read_all_text path
  end

  def self.read_all_bytes(path)
    @@api.read_all_bytes path
  end

  def self.write_all_text(path, text)
    @@api.write_all_text path, text
  end

  def self.write_all_bytes(path, buff)
    @@api.write_all_bytes path, buff
  end
end
