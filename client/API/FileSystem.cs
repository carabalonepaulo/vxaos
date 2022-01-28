using System;
using Godot;
using IronRuby.Builtins;

namespace API
{
    public class FileSystem
    {
        public bool Exists(string path)
        {
            var file = new File();
            return file.FileExists(path);
        }

        public string ReadAllText(string path)
        {
            var file = new File();
            file.Open(path, File.ModeFlags.Read);
            var content = file.GetAsText();
            file.Close();
            return content;
        }

        public MutableString ReadAllBytes(string path)
        {
            var file = new File();
            file.Open(path, File.ModeFlags.Read);

            var len = (int)file.GetLen();
            var content = file.GetBuffer(len);
            var buff = MutableString.CreateBinary(content);

            file.Close();
            return buff;
        }

        public void WriteAllText(string path, string text)
        {

        }

        public void WriteAllBytes(string path, MutableString buff)
        {

        }
    }
}