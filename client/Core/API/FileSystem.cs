using Godot;
using IronRuby.Builtins;

namespace API
{
    public class FileSystem
    {
        public FileSystem()
        {

        }

        public bool exists(string path)
        {
            var file = new File();
            return file.FileExists(path);
        }

        public void write_all_text(string path, string text)
        {

        }

        public void write_all_bytes(string path, byte[] buff)
        {

        }

        public string read_all_text(string path)
        {
            var file = new File();
            file.Open(path, File.ModeFlags.Read);
            var content = file.GetAsText();
            file.Close();
            return content;
        }

        public MutableString read_all_bytes(string path)
        {
            var file = new File();
            file.Open(path, File.ModeFlags.Read);

            var len = (int)file.GetLen();
            var content = file.GetBuffer(len);
            var buff = MutableString.CreateBinary(content);

            file.Close();
            return buff;
        }
    }
}
