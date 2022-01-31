load_assembly 'IronRuby.Libraries', 'IronRuby.StandardLibrary.Zlib'

class RGSS
  ScriptsFileNotFound = 'Scripts.rvdata2 not found.'
  ScriptsFilePath = './Data/Scripts.rvdata2'

  def inflate(code)
    Zlib::Inflate.inflate(code).force_encoding('UTF-8')
  end

  def load_scripts
    raise ScriptsFileNotFound unless $file_system.exists(ScriptsFilePath)
    buff = $file_system.read_all_bytes ScriptsFilePath
    Marshal.load(buff).each { |script| eval inflate(script[2]) }
  end
end